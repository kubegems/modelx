package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-logr/logr"
	"kubegems.io/modelx/pkg/errors"
	"kubegems.io/modelx/pkg/types"
)

var GlobalExtensions = map[string]Extension{}

type Extension interface {
	Download(ctx context.Context, blob types.Descriptor, location types.BlobLocation, into io.Writer) error
	Upload(ctx context.Context, blob DescriptorWithContent, location types.BlobLocation) error
}

func NewDelegateExtension() *DelegateExtension {
	return &DelegateExtension{
		Extensions: GlobalExtensions,
	}
}

type DelegateExtension struct {
	Extensions map[string]Extension
}

func (e DelegateExtension) Download(ctx context.Context, blob types.Descriptor, location types.BlobLocation, into io.Writer) error {
	log := logr.FromContextOrDiscard(ctx).WithValues(
		"provider", location.Provider,
		"properties", location.Properties)
	log.Info("extend downloading blob")

	if ext, ok := e.Extensions[location.Provider]; ok {
		return ext.Download(ctx, blob, location, into)
	}
	return errors.NewUnsupportedError("provider: " + location.Provider)
}

func (e DelegateExtension) Upload(ctx context.Context, blob DescriptorWithContent, location types.BlobLocation) error {
	log := logr.FromContextOrDiscard(ctx).WithValues(
		"provider", location.Provider,
		"properties", location.Properties)
	log.Info("extend uploading blob")
	if ext, ok := e.Extensions[location.Provider]; ok {
		return ext.Upload(ctx, blob, location)
	}
	return errors.NewUnsupportedError("provider: " + location.Provider)
}

type BlobContent struct {
	Content       io.ReadCloser
	ContentLength int64
}

func (t *RegistryClient) extrequest(ctx context.Context, method, url string, header map[string][]string, contentlen int64, content io.ReadCloser) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, t.Registry+url, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range header {
		req.Header[k] = v
	}
	req.Header.Set("Authorization", t.Authorization)
	req.Header.Set("User-Agent", UserAgent)
	req.Body = content
	req.ContentLength = contentlen

	norediretccli := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // do not follow redirect
		},
	}

	resp, err := norediretccli.Do(req)
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

func convertProperties(dest any, src any) error {
	raw, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, dest)
}
