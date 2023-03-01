package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func HTTPDownload(ctx context.Context, location *url.URL, into io.Writer) error {
	req, err := http.NewRequest("GET", location.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", UserAgent)
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

func HTTPUpload(ctx context.Context, location *url.URL, blob DescriptorWithContent) error {
	content, err := blob.GetContent()
	if err != nil {
		return err
	}
	// s3 upload use PUT
	method := http.MethodPost
	if location.Query().Has("X-Amz-Credential") {
		method = http.MethodPut
	}
	req, err := http.NewRequest(method, location.String(), content)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("User-Agent", UserAgent)
	req.ContentLength = blob.Size
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}
	return nil
}
