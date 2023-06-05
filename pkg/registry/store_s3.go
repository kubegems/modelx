package registry

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/opencontainers/go-digest"
	modelxerrors "kubegems.io/modelx/pkg/errors"

	"kubegems.io/modelx/pkg/types"
)

const (
	MultiPartUploadThreshold = 5 * 1024 * 1024 * 1024
	DefaultPartCount         = 3 // parts if no size
)

var ErrUploadNotFound = modelxerrors.NewInternalError(errors.New("upload not found"))

type S3RegistryStore struct {
	fs       RegistryStore
	provider *S3StorageProvider
}

var _ RegistryStore = &S3RegistryStore{}

func NewS3RegistryStore(ctx context.Context, options *Options) (*S3RegistryStore, error) {
	fs, err := NewS3FSProvider(ctx, options.S3)
	if err != nil {
		return nil, err
	}
	store := &FSRegistryStore{
		FS:             fs,
		EnableRedirect: options.EnableRedirect,
	}
	if err := store.RefreshGlobalIndex(ctx); err != nil {
		return nil, err
	}
	return &S3RegistryStore{fs: store, provider: fs}, nil
}

func (s *S3RegistryStore) GetGlobalIndex(ctx context.Context, search string) (types.Index, error) {
	return s.fs.GetGlobalIndex(ctx, search)
}

func (s *S3RegistryStore) GetIndex(ctx context.Context, repository string, search string) (types.Index, error) {
	return s.fs.GetIndex(ctx, repository, search)
}

func (s *S3RegistryStore) RemoveIndex(ctx context.Context, repository string) error {
	return s.fs.RemoveIndex(ctx, repository)
}

func (s *S3RegistryStore) ExistsManifest(ctx context.Context, repository string, reference string) (bool, error) {
	return s.fs.ExistsManifest(ctx, repository, reference)
}

func (s *S3RegistryStore) GetManifest(ctx context.Context, repository string, reference string) (*types.Manifest, error) {
	return s.fs.GetManifest(ctx, repository, reference)
}

func (s *S3RegistryStore) PutManifest(ctx context.Context, repository string, reference string, contentType string, manifest types.Manifest) error {
	// complete multipart upload
	for _, blob := range manifest.Blobs {
		path := BlobDigestPath(repository, blob.Digest)
		if err := s.completeMultipartUpload(ctx, path); err != nil {
			return err
		}
	}
	return s.fs.PutManifest(ctx, repository, reference, contentType, manifest)
}

func (s *S3RegistryStore) DeleteManifest(ctx context.Context, repository string, reference string) error {
	return s.fs.DeleteManifest(ctx, repository, reference)
}

func (s *S3RegistryStore) ListBlobs(ctx context.Context, repository string) ([]digest.Digest, error) {
	return s.fs.ListBlobs(ctx, repository)
}

func (s *S3RegistryStore) GetBlob(ctx context.Context, repository string, digest digest.Digest) (*BlobContent, error) {
	return s.fs.GetBlob(ctx, repository, digest)
}

func (s *S3RegistryStore) DeleteBlob(ctx context.Context, repository string, digest digest.Digest) error {
	return s.fs.DeleteBlob(ctx, repository, digest)
}

func (s *S3RegistryStore) PutBlob(ctx context.Context, repository string, digest digest.Digest, content BlobContent) error {
	return s.fs.PutBlob(ctx, repository, digest, content)
}

func (s *S3RegistryStore) ExistsBlob(ctx context.Context, repository string, digest digest.Digest) (bool, error) {
	return s.fs.ExistsBlob(ctx, repository, digest)
}

func (s *S3RegistryStore) GetBlobLocation(ctx context.Context, repository string, digest digest.Digest,
	purpose string, properties map[string]string,
) (*BlobLocation, error) {
	path := BlobDigestPath(repository, digest)
	switch purpose {
	case BlobLocationPurposeDownload:
		return s.downloadLocation(ctx, path, properties)
	case BlobLocationPurposeUpload:
		return s.uploadLocation(ctx, path, properties)
	default:
		return nil, modelxerrors.NewUnsupportedError("purpose: " + purpose)
	}
}

func (s *S3RegistryStore) completeMultipartUpload(ctx context.Context, path string) error {
	uploadid, err := s.getUploadId(ctx, path, false)
	if err != nil {
		if err != ErrUploadNotFound {
			return err
		}
	}
	if uploadid == nil {
		return nil
	}
	// list parts
	listparts := &s3.ListPartsInput{
		Bucket:   aws.String(s.provider.Bucket),
		Key:      s.provider.prefixedKey(path),
		UploadId: uploadid,
	}
	listpartsOutput, err := s.provider.Client.ListParts(ctx, listparts)
	if err != nil {
		return err
	}

	complete := &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(s.provider.Bucket),
		Key:      s.provider.prefixedKey(path),
		UploadId: uploadid,
		MultipartUpload: &s3types.CompletedMultipartUpload{
			Parts: func() []s3types.CompletedPart {
				parts := make([]s3types.CompletedPart, len(listpartsOutput.Parts))
				for i, part := range listpartsOutput.Parts {
					parts[i] = s3types.CompletedPart{
						ETag:       part.ETag,
						PartNumber: part.PartNumber,
					}
				}
				return parts
			}(),
		},
	}
	_, err = s.provider.Client.CompleteMultipartUpload(ctx, complete)
	if err != nil {
		return err
	}
	return nil
}

func (s *S3RegistryStore) uploadLocation(
	ctx context.Context, path string, properties map[string]string,
) (*BlobLocation, error) {
	if properties == nil {
		properties = make(map[string]string)
	}
	size, _ := strconv.Atoi(properties["size"])
	usemultipart, _ := strconv.ParseBool(properties["multipart"])
	name := properties["name"]
	if usemultipart || size > MultiPartUploadThreshold {
		return s.uploadLocationMultiPart(ctx, path, size, properties)
	}
	putobj := &s3.PutObjectInput{
		Bucket: aws.String(s.provider.Bucket),
		Key:    s.provider.prefixedKey(path),
		Metadata: map[string]string{
			"FileName": name, // save file name in metadata
		},
	}
	out, err := s.provider.PreSign.PresignPutObject(ctx, putobj, s3.WithPresignExpires(s.provider.Expire))
	if err != nil {
		return nil, err
	}
	return &BlobLocation{
		Provider: "s3",
		Purpose:  BlobLocationPurposeUpload,
		Properties: map[string]any{
			"parts": []presignedPart{{
				URL:          out.URL,
				Method:       out.Method,
				SignedHeader: out.SignedHeader,
			}},
		},
	}, nil
}

type presignedPart struct {
	URL          string              `json:"url,omitempty"`
	Method       string              `json:"method,omitempty"`
	SignedHeader map[string][]string `json:"signedHeader,omitempty"`
	PartNumber   int                 `json:"partNumber,omitempty"`
}

func (s *S3RegistryStore) getUploadId(ctx context.Context, path string, withCreate bool) (*string, error) {
	key := s.provider.prefixedKey(path)
	input := &s3.CreateMultipartUploadInput{
		Bucket:  aws.String(s.provider.Bucket),
		Key:     key,
		Expires: aws.Time(time.Now().Add(s.provider.Expire)),
	}
	existsupload, err := s.provider.Client.ListMultipartUploads(ctx, &s3.ListMultipartUploadsInput{
		Bucket:    aws.String(s.provider.Bucket),
		Delimiter: aws.String("/"),
		Prefix:    key,
	})
	if err != nil {
		return nil, err
	}
	var uploadid *string
	if len(existsupload.Uploads) > 0 {
		uploadid = existsupload.Uploads[0].UploadId
	} else {
		if !withCreate {
			return nil, ErrUploadNotFound
		}
		createOutput, err := s.provider.Client.CreateMultipartUpload(ctx, input)
		if err != nil {
			return nil, err
		}
		uploadid = createOutput.UploadId
	}
	return uploadid, nil
}

func (s *S3RegistryStore) uploadLocationMultiPart(
	ctx context.Context, path string, size int, properties map[string]string,
) (*BlobLocation, error) {
	uploadid, err := s.getUploadId(ctx, path, true)
	if err != nil {
		return nil, err
	}
	partsCount := DefaultPartCount
	if count := size / MultiPartUploadThreshold; count != 0 {
		if size%MultiPartUploadThreshold != 0 {
			count++
		}
		partsCount = count
	}
	presignedParts := make([]presignedPart, partsCount)
	for i := 0; i < partsCount; i++ {
		partNumber := i + 1
		presignUploadPart := &s3.UploadPartInput{
			Bucket:     aws.String(s.provider.Bucket),
			Key:        s.provider.prefixedKey(path),
			UploadId:   uploadid,
			PartNumber: int32(partNumber), // [1,10000]
		}
		req, err := s.provider.PreSign.PresignUploadPart(ctx, presignUploadPart, s3.WithPresignExpires(s.provider.Expire))
		if err != nil {
			return nil, err
		}
		presignedParts[i] = presignedPart{
			URL:          req.URL,
			Method:       req.Method,
			SignedHeader: req.SignedHeader,
			PartNumber:   partNumber,
		}
	}
	return &BlobLocation{
		Provider: "s3",
		Purpose:  BlobLocationPurposeUpload,
		Properties: map[string]any{
			"multipart": true,
			"uploadId":  uploadid,
			"parts":     presignedParts,
		},
	}, nil
}

func (s *S3RegistryStore) downloadLocation(
	ctx context.Context, path string, properties map[string]string,
) (*BlobLocation, error) {
	getobj := &s3.GetObjectInput{
		Bucket: aws.String(s.provider.Bucket),
		Key:    s.provider.prefixedKey(path),
	}
	out, err := s.provider.PreSign.PresignGetObject(ctx, getobj, s3.WithPresignExpires(s.provider.Expire))
	if err != nil {
		return nil, err
	}
	return &BlobLocation{
		Provider: "s3",
		Purpose:  BlobLocationPurposeDownload,
		Properties: map[string]any{
			"parts": []presignedPart{{
				URL:          out.URL,
				Method:       out.Method,
				SignedHeader: out.SignedHeader,
			}},
		},
	}, nil
}
