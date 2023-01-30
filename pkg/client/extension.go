package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/opencontainers/go-digest"
	"kubegems.io/modelx/pkg/errors"
	"kubegems.io/modelx/pkg/types"
)

var GlobalExtensions = map[string]Extension{
	"idoe":  &IdoeExt{},
	"idoes": &IdoeExt{},
}

type Extension interface {
	Download(ctx context.Context, location *url.URL, into io.Writer) error
	Upload(ctx context.Context, location *url.URL, blob *BlobContent) error
}

func ExtDownload(ctx context.Context, location *url.URL, into io.Writer) error {
	if ext, ok := GlobalExtensions[location.Scheme]; ok {
		return ext.Download(ctx, location, into)
	}
	return HTTPDownload(ctx, location, into)
}

func ExtUpload(ctx context.Context, location *url.URL, blob *BlobContent) error {
	if ext, ok := GlobalExtensions[location.Scheme]; ok {
		return ext.Upload(ctx, location, blob)
	}
	return HTTPUpload(ctx, location, blob)
}

type BlobContent struct {
	Content       io.ReadSeekCloser
	ContentLength int64
}

func (t *RegistryClient) HeadBlob(ctx context.Context, repository string, digest digest.Digest) (bool, error) {
	path := "/" + repository + "/blobs/" + digest.String()
	resp, err := t.request(ctx, "HEAD", path, nil, "", nil, nil)
	if err != nil {
		return false, err
	}
	return resp.StatusCode == http.StatusOK, nil
}

func (t *RegistryClient) GetBlobContent(ctx context.Context, repository string, digest digest.Digest, into io.Writer) error {
	path := "/" + repository + "/blobs/" + digest.String()
	headers := map[string][]string{}
	resp, err := t.requestRaw(ctx, "GET", path, headers, nil)
	if err != nil {
		return err
	}
	if http.StatusMultipleChoices <= resp.StatusCode && resp.StatusCode < http.StatusBadRequest {
		loc := resp.Header.Get("Location")
		if loc == "" {
			return errors.NewInternalError(fmt.Errorf("no Location header found in a %s response", resp.Status))
		}
		locau, err := url.Parse(loc)
		if err != nil {
			return errors.NewInternalError(fmt.Errorf("invalid location %s: %w", loc, err))
		}
		return ExtDownload(ctx, locau, into)
	}

	_, err = io.CopyN(into, resp.Body, resp.ContentLength)
	return err
}

func (t *RegistryClient) UploadBlobContent(ctx context.Context, repository string, desc types.Descriptor, blob BlobContent) error {
	header := map[string][]string{
		"Content-Type": {"application/octet-stream"},
	}
	path := "/" + repository + "/blobs/" + desc.Digest.String()

	resp, err := t.requestRaw(ctx, "PUT", path, header, &blob)
	if err != nil {
		return err
	}
	if http.StatusMultipleChoices <= resp.StatusCode && resp.StatusCode < http.StatusBadRequest {
		loc := resp.Header.Get("Location")
		if loc == "" {
			return errors.NewInternalError(fmt.Errorf("no Location header found in a %s response", resp.Status))
		}
		locau, err := url.Parse(loc)
		if err != nil {
			return errors.NewInternalError(fmt.Errorf("invalid location %s: %w", loc, err))
		}
		return ExtUpload(ctx, locau, &blob)
	}
	return nil
}

func (t *RegistryClient) requestRaw(ctx context.Context, method, url string, header map[string][]string, blob *BlobContent) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, t.Registry+url, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range header {
		req.Header[k] = v
	}
	req.Header.Set("Authorization", t.Authorization)
	req.Header.Set("User-Agent", UserAgent)

	if blob != nil {
		req.Body = blob.Content
		req.ContentLength = blob.ContentLength
	}
	resp, err := t.httpcli.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest && req.Method != "HEAD" {
		var apierr errors.ErrorInfo
		if resp.Header.Get("Content-Type") == "application/json" {
			if err := json.NewDecoder(resp.Body).Decode(&apierr); err != nil {
				return nil, err
			}
		} else {
			bodystr, _ := io.ReadAll(resp.Body)
			apierr.Message = string(bodystr)
		}
		apierr.HttpStatus = resp.StatusCode
		return nil, apierr
	}
	return resp, nil
}
