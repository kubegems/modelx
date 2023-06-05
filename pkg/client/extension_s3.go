package client

import (
	"context"
	"fmt"
	"io"
	"net/url"

	"golang.org/x/sync/errgroup"
	"kubegems.io/modelx/pkg/types"
)

func init() {
	GlobalExtensions["s3"] = S3Extension{}
}

const (
	UploadPartConcurrency   = 3
	DownloadPartConcurrency = 3
)

type S3Extension struct{}

func (e S3Extension) Download(ctx context.Context, blob types.Descriptor, location types.BlobLocation, into io.Writer) error {
	var properties S3Properties
	convertProperties(&properties, location.Properties)

	if len(properties.Parts) == 0 {
		return fmt.Errorf("no parts found")
	}
	firstpart := properties.Parts[0]
	u, err := url.Parse(firstpart.URL)
	if err != nil {
		return err
	}
	return HTTPDownload(ctx, u, firstpart.SignedHeader, into)
}

type S3Properties struct {
	Multipart bool            `json:"multipart,omitempty"`
	UploadID  string          `json:"uploadID,omitempty"`
	Parts     []presignedPart `json:"parts,omitempty"`
}

type presignedPart struct {
	URL          string              `json:"url,omitempty"`
	Method       string              `json:"method,omitempty"`
	SignedHeader map[string][]string `json:"signedHeader,omitempty"`
	PartNumber   int                 `json:"partNumber,omitempty"`
}

func (e S3Extension) Upload(ctx context.Context, blob DescriptorWithContent, location types.BlobLocation) error {
	var properties S3Properties
	convertProperties(&properties, location.Properties)

	parts := calcParts(blob.Size, len(properties.Parts))
	for i := range parts {
		u, err := url.Parse(properties.Parts[i].URL)
		if err != nil {
			return err
		}
		parts[i].url = u
		parts[i].header = properties.Parts[i].SignedHeader
	}
	eg := errgroup.Group{}
	eg.SetLimit(UploadPartConcurrency)
	for i := range parts {
		part := parts[i]
		eg.Go(func() error {
			return retry(ctx, 3, func() error {
				getoffsetbody := func() (io.ReadCloser, error) {
					partcontent, err := blob.GetContent()
					if err != nil {
						return nil, err
					}
					n, err := partcontent.Seek(part.offset, io.SeekStart)
					if err != nil {
						return nil, err
					}
					_ = n
					limitbody := NewSectionReader(partcontent, part.offset, part.length)
					return limitbody, nil
				}
				return HTTPUpload(ctx, part.url, part.header, part.length, getoffsetbody)
			})
		})
	}
	return eg.Wait()
}

type PartRange struct {
	url    *url.URL
	header map[string][]string
	offset int64
	length int64
	w      io.WriterAt
}

func calcParts(total int64, partscount int) []PartRange {
	partsize := total / int64(partscount)

	parts := make([]PartRange, partscount)
	for i := range parts {
		parts[i].offset = int64(i) * partsize
		if i == len(parts)-1 {
			parts[i].length = total - parts[i].offset // last part
		} else {
			parts[i].length = partsize
		}
	}
	return parts
}

// NewSectionReader returns a SectionReader that reads from r
// starting at offset off and stops with EOF after n bytes.
// it's a copy from io.NewSectionReader but use io.ReadSeekCloser and seek to offset on init
func NewSectionReader(rc io.ReadSeekCloser, offset, length int64) io.ReadCloser {
	_, err := rc.Seek(offset, io.SeekStart)
	if err != nil {
		panic(err)
	}
	return &ReadCloser{
		Reader: io.LimitReader(rc, length),
		Closer: rc,
	}
}

type ReadCloser struct {
	io.Reader
	io.Closer
}

func retry(ctx context.Context, max int, fn func() error) error {
	var reterr error
	for i := 0; i < max; i++ {
		if err := fn(); err != nil {
			reterr = err
		} else {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
	return reterr
}
