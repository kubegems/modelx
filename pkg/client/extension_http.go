package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func HTTPDownload(ctx context.Context, location *url.URL, header http.Header, into io.Writer) error {
	req, err := http.NewRequestWithContext(ctx, "GET", location.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", UserAgent)
	for k, v := range header {
		req.Header[k] = v
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}
	_, err = io.Copy(into, resp.Body)
	return err
}

func HTTPUpload(ctx context.Context, location *url.URL, header http.Header, contentlen int64, getbody func() (io.ReadCloser, error)) error {
	method := http.MethodPost
	// s3 upload use PUT
	if location.Query().Has("X-Amz-Credential") {
		method = http.MethodPut
	}
	body, err := getbody()
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, location.String(), body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("User-Agent", UserAgent)
	for k, v := range header {
		req.Header[k] = v
	}
	req.ContentLength, req.GetBody = contentlen, getbody

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %s %s", resp.Status, body)
	}
	return nil
}
