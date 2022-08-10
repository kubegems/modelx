package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/opencontainers/go-digest"
	"kubegems.io/modelx/pkg/errors"
	"kubegems.io/modelx/pkg/types"
)

type RegistryClient struct {
	Client        *http.Client
	Registry      string
	Authorization string
}

func (t *RegistryClient) UploadBlob(ctx context.Context, repository string, desc types.Descriptor, getbody GetBodyFunc) error {
	header := map[string]string{
		"Content-Type":   "application/octet-stream",
		"Content-Length": strconv.FormatInt(desc.Size, 10),
	}
	path := "/" + repository + "/blobs/" + desc.Digest.String()

	resp, err := t.request(ctx, "PUT", path, header, getbody, nil)
	if err != nil {
		return err
	}
	_ = resp
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

	body, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	getbody := func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	if _, err := t.request(ctx, "PUT", path, header, getbody, nil); err != nil {
		return err
	}
	return nil
}

func (t *RegistryClient) GetIndex(ctx context.Context, repository string, search string) (*types.Index, error) {
	index := &types.Index{}
	path := "/" + repository + "/index" + "?search=" + search
	_, err := t.request(ctx, "GET", path, nil, nil, index)
	if err != nil {
		return nil, err
	}
	return index, nil
}

func (t *RegistryClient) GetGlobalIndex(ctx context.Context, search string) (*types.Index, error) {
	query := url.Values{}
	if search != "" {
		query.Add("search", search)
	}
	path := "/"
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	index := &types.Index{}
	_, err := t.request(ctx, "GET", path, nil, nil, index)
	if err != nil {
		return nil, err
	}
	return index, nil
}

type GetBodyFunc func() (io.ReadCloser, error)

func (t *RegistryClient) request(ctx context.Context, method, url string, header map[string]string, getbody GetBodyFunc, into any) (*http.Response, error) {
	applyreqfuncs := []func(req *http.Request){}

	if len(header) > 0 {
		applyreqfuncs = append(applyreqfuncs, func(req *http.Request) {
			for k, v := range header {
				req.Header.Set(k, v)
			}
		})
	}

	var reqbody io.Reader
	if getbody != nil {
		body, err := getbody()
		if err != nil {
			return nil, err
		}
		reqbody = body
		// In order to http.Client can resolve redirect when body is not empty, a GetBodyFunc must be set.
		// http.Client use GetBody to get the a new body for the next redirect request.
		applyreqfuncs = append(applyreqfuncs, func(req *http.Request) {
			if req.GetBody == nil {
				req.GetBody = getbody
			}
		})
	}

	req, err := http.NewRequestWithContext(ctx, method, t.Registry+url, reqbody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", t.Authorization)

	for _, f := range applyreqfuncs {
		f(req)
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
		apierr.HttpStatus = resp.StatusCode
		return nil, apierr
	}
	if into != nil {
		if err := json.NewDecoder(resp.Body).Decode(into); err != nil {
			return nil, err
		}
	}
	return resp, nil
}
