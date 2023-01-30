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

func HTTPUpload(ctx context.Context, location *url.URL, blob *BlobContent) error {
	req, err := http.NewRequest("POST", location.String(), blob.Content)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = blob.ContentLength
	req.Header.Set("User-Agent", UserAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}
	return nil
}
