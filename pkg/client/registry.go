package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/opencontainers/go-digest"
	"kubegems.io/modelx/pkg/errors"
	"kubegems.io/modelx/pkg/types"
)

type RegistryClient struct {
	Client *http.Client
	Addr   string
}

func (t *RegistryClient) UploadBlob(ctx context.Context, repository string, desc types.Descriptor, body io.Reader) error {
	header := map[string]string{
		"Content-Type": "application/octet-stream",
	}
	path := "/" + repository + "/blobs/" + desc.Digest.String()
	if _, err := t.request(ctx, "PUT", path, header, body, nil); err != nil {
		return err
	}
	return nil
}

func (t *RegistryClient) GetBlob(ctx context.Context, repository string, digest digest.Digest) (io.ReadCloser, int64, error) {
	path := "/" + repository + "/blobs/" + digest.String()
	resp, err := t.request(ctx, "GET", path, nil, nil, nil)
	if err != nil {
		return nil, -1, err
	}
	return resp.Body, resp.ContentLength, nil
}

func (t *RegistryClient) HeadBlob(ctx context.Context, repository string, digest digest.Digest) (bool, error) {
	path := "/" + repository + "/blobs/" + digest.String()
	resp, err := t.request(ctx, "HEAD", path, nil, nil, nil)
	if err != nil {
		return false, err
	}
	return resp.StatusCode == http.StatusOK, nil
}

func (t *RegistryClient) GetManifest(ctx context.Context, repository string, version string) (*types.Manifest, error) {
	if version == "" {
		version = "latest"
	}
	manifest := &types.Manifest{}
	path := "/" + repository + "/manifests/" + version
	_, err := t.request(ctx, "GET", path, nil, nil, manifest)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

func (t *RegistryClient) PutManifest(ctx context.Context, repository string, version string, manifest types.Manifest) error {
	if version == "" {
		version = "latest"
	}

	header := map[string]string{
		"Content-Type": "application/json",
	}
	path := "/" + repository + "/manifests/" + version
	_, err := t.request(ctx, "PUT", path, header, manifest, nil)
	if err != nil {
		return err
	}
	return nil
}

func (t *RegistryClient) GetIndex(ctx context.Context, repository string, search string) (*types.Index, error) {
	index := &types.Index{}
	path := "/" + repository + "/manifests" + "?search=" + search
	_, err := t.request(ctx, "GET", path, nil, nil, index)
	if err != nil {
		return nil, err
	}
	return index, nil
}

func (t *RegistryClient) GetGlobalIndex(ctx context.Context, search string) (*types.GlobalIndex, error) {
	index := &types.GlobalIndex{}
	path := "/" + "?" + url.Values{"search": {search}}.Encode()
	_, err := t.request(ctx, "GET", path, nil, nil, index)
	if err != nil {
		return nil, err
	}
	return index, nil
}

func (t *RegistryClient) request(ctx context.Context, method, url string, header map[string]string, body any, into any) (*http.Response, error) {
	url = t.Addr + url

	var reqbody io.Reader
	switch val := body.(type) {
	case io.Reader:
		reqbody = val
	case nil:
		reqbody = nil
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return nil, err
		}
		reqbody = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reqbody)
	if err != nil {
		return nil, err
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	resp, err := t.Client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 && req.Method != "HEAD" {
		var apierr errors.ErrorInfo
		if resp.Header.Get("Content-Type") == "application/json" {
			if err := json.NewDecoder(resp.Body).Decode(&apierr); err != nil {
				return nil, err
			}
		} else {
			bodystr, _ := io.ReadAll(resp.Body)
			apierr.Message = string(bodystr)
		}
		return nil, apierr
	}
	if into != nil {
		if err := json.NewDecoder(resp.Body).Decode(into); err != nil {
			return nil, err
		}
	}
	return resp, nil
}
