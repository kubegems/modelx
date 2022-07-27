package registry

import (
	"context"
	"errors"
	"io"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/transport/http"
)

type StorageContent struct {
	Content       io.ReadCloser
	ContentLength int64
	ContentType   string
}

func (s StorageContent) Close() error {
	if s.Content != nil {
		return s.Content.Close()
	}
	return nil
}

func (s StorageContent) Read(p []byte) (int, error) {
	return s.Content.Read(p)
}

type S3StorageProvider struct {
	Bucket  string
	Client  *s3.Client
	PreSign *s3.PresignClient
	Expire  time.Duration
	Prefix  string
}

func (m *S3StorageProvider) Put(ctx context.Context, path string, content StorageContent) error {
	uploadobj := &s3.PutObjectInput{
		Bucket:        aws.String(m.Bucket),
		Key:           m.prefixedKey(path),
		Body:          content.Content,
		ContentLength: int64(content.ContentLength),
		ContentType:   aws.String(content.ContentType),
	}
	uploader := manager.NewUploader(m.Client)
	uploadout, err := uploader.Upload(ctx, uploadobj)
	if err != nil {
		return err
	}
	_ = uploadout
	return nil
}

func (m *S3StorageProvider) Remove(ctx context.Context, path string) error {
	_, err := m.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(m.Bucket),
		Key:    m.prefixedKey(path),
	})
	return err
}

func (m *S3StorageProvider) Get(ctx context.Context, path string) (StorageContent, error) {
	getobjout, err := m.Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(m.Bucket),
		Key:    m.prefixedKey(path),
	})
	if err != nil {
		return StorageContent{}, err
	}
	return StorageContent{
		Content:       getobjout.Body,
		ContentType:   *getobjout.ContentType,
		ContentLength: getobjout.ContentLength,
	}, nil
}

func (m *S3StorageProvider) Exists(ctx context.Context, path string) (bool, error) {
	_, err := m.Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(m.Bucket),
		Key:    m.prefixedKey(path),
	})
	if err != nil {
		if IsStorageNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

type StorageMeta struct {
	Name         string
	Size         int64
	LastModified time.Time
	Metadata     map[string]string
}

func (m *S3StorageProvider) List(ctx context.Context, path string, isPrefix bool) ([]StorageMeta, error) {
	prefix := *m.prefixedKey(path)
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	listinput := &s3.ListObjectsInput{
		Bucket: aws.String(m.Bucket),
		Prefix: aws.String(prefix),
	}
	if !isPrefix {
		listinput.Delimiter = aws.String("/")
	}
	var result []StorageMeta
	listobjout, err := m.Client.ListObjects(ctx, listinput)
	if err != nil {
		return nil, err
	}
	for _, obj := range listobjout.Contents {
		result = append(result, StorageMeta{
			Name:         strings.TrimPrefix(*obj.Key, prefix),
			Size:         obj.Size,
			LastModified: *obj.LastModified,
		})
	}
	for listobjout.IsTruncated {
		listinput.Marker = listobjout.NextMarker
		listobjout, err = m.Client.ListObjects(ctx, listinput)
		if err != nil {
			return nil, err
		}
		for _, obj := range listobjout.Contents {
			result = append(result, StorageMeta{
				Name:         strings.TrimPrefix(*obj.Key, prefix),
				Size:         obj.Size,
				LastModified: *obj.LastModified,
			})
		}
	}
	return result, nil
}

func (m *S3StorageProvider) GetURL(ctx context.Context, path string) (string, error) {
	getobj := &s3.GetObjectInput{
		Bucket: aws.String(m.Bucket),
		Key:    m.prefixedKey(path),
	}
	out, err := m.PreSign.PresignGetObject(ctx, getobj, s3.WithPresignExpires(m.Expire))
	if err != nil {
		return "", err
	}
	return out.URL, nil
}

func (m *S3StorageProvider) UploadURL(ctx context.Context, path string) (string, error) {
	putobj := &s3.PutObjectInput{
		Bucket: aws.String(m.Bucket),
		Key:    m.prefixedKey(path),
	}
	out, err := m.PreSign.PresignPutObject(ctx, putobj, s3.WithPresignExpires(m.Expire))
	if err != nil {
		return "", err
	}
	return out.URL, nil
}

func IsStorageNotFound(err error) bool {
	var apie *http.ResponseError
	if errors.As(err, &apie) {
		return apie.HTTPStatusCode() == 404
	}
	return false
}

func (m *S3StorageProvider) prefixedKey(key string) *string {
	return aws.String(path.Join(m.Prefix, key))
}
