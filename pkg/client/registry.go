package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"kubegems.io/modelx/pkg/errors"
	"kubegems.io/modelx/pkg/types"
	"kubegems.io/modelx/pkg/version"
)

var UserAgent = "modelx/" + version.Get().GitVersion

func NewRegistryClient(addr string, auth string) *RegistryClient {
	return &RegistryClient{
		Registry:      addr,
		Authorization: auth,
	}
}

type RegistryClient struct {
	Registry      string
	Authorization string
}

func (t *RegistryClient) GetManifest(ctx context.Context, repository string, version string) (*types.Manifest, error) {
	if version == "" {
		version = "latest"
	}
	manifest := &types.Manifest{}
	path := "/" + repository + "/manifests/" + version
	if err := t.simplerequest(ctx, "GET", path, manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

func (t *RegistryClient) PutManifest(ctx context.Context, repository string, version string, manifest types.Manifest) error {
	if version == "" {
		version = "latest"
	}
	path := "/" + repository + "/manifests/" + version
	data, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	return t.simpleuploadrequest(ctx, "PUT", path, "application/json", data, nil)
}

func (t *RegistryClient) GetIndex(ctx context.Context, repository string, search string) (*types.Index, error) {
	index := &types.Index{}
	path := "/" + repository + "/index" + "?search=" + search
	if err := t.simplerequest(ctx, "GET", path, index); err != nil {
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
	if err := t.simplerequest(ctx, "GET", path, index); err != nil {
		return nil, err
	}
	return index, nil
}

type GetContentFunc func() (io.ReadSeekCloser, error)

type RqeuestBody struct {
	ContentLength int64
	ContentBody   func() (io.ReadSeekCloser, error)
}

func (t *RegistryClient) simplerequest(ctx context.Context, method, url string, into any) error {
	_, err := t.request(ctx, method, url, nil, "", nil, into)
	return err
}

func (t *RegistryClient) simpleuploadrequest(ctx context.Context, method, url string, contenttype string, contentdata []byte, into any) error {
	_, err := t.request(ctx, method, url, nil, contenttype, contentdata, into)
	return err
}

func (t *RegistryClient) request(ctx context.Context, method, url string, header map[string]string, contenttype string, postdata []byte, into any) (*http.Response, error) {
	var reqbody io.Reader
	if postdata != nil {
		reqbody = bytes.NewReader(postdata)
	}
	req, err := http.NewRequestWithContext(ctx, method, t.Registry+url, reqbody)
	if err != nil {
		return nil, err
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	if contenttype != "" {
		req.Header.Set("Content-Type", contenttype)
	}
	req.Header.Set("Authorization", t.Authorization)
	req.Header.Set("User-Agent", UserAgent)

	resp, err := http.DefaultClient.Do(req)
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
