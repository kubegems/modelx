package registry

import (
	"context"
	"time"
)

type FsObjectMeta struct {
	Name         string
	Size         int64
	LastModified time.Time
	ContentType  string
}

type FSProvider interface {
	Put(ctx context.Context, path string, content BlobContent) error
	Get(ctx context.Context, path string) (*BlobContent, error)
	Stat(ctx context.Context, path string) (FsObjectMeta, error)
	Remove(ctx context.Context, path string, recursive bool) error
	Exists(ctx context.Context, path string) (bool, error)
	List(ctx context.Context, path string, recursive bool) ([]FsObjectMeta, error)
}

func (s BlobContent) Close() error {
	if s.Content != nil {
		return s.Content.Close()
	}
	return nil
}

func (s BlobContent) Read(p []byte) (int, error) {
	return s.Content.Read(p)
}

func StringDeref(ptr *string, def string) string {
	if ptr != nil {
		return *ptr
	}
	return def
}

func TimeDeref(ptr *time.Time, def time.Time) time.Time {
	if ptr != nil {
		return *ptr
	}
	return def
}
