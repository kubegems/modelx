package registry

import (
	"context"
	"encoding/json"
	"io"
	"os"
	iopath "path"
	"path/filepath"
	"strings"

	modelxerrors "kubegems.io/modelx/pkg/errors"
)

const (
	DefaultFileMode = 0o644
	DefaultDirMode  = 0o755
)

type LocalFSOptions struct {
	Basepath string
}

func NewDefaultLocalFSOptions() *LocalFSOptions {
	return &LocalFSOptions{
		Basepath: "data/registry",
	}
}

var _ FSProvider = &LocalFSProvider{}

type LocalFSProvider struct {
	basepath string
}

func NewLocalFSProvider(options *LocalFSOptions) (*LocalFSProvider, error) {
	if err := os.MkdirAll(options.Basepath, DefaultDirMode); err != nil {
		return nil, err
	}
	return &LocalFSProvider{basepath: options.Basepath}, nil
}

type localFileMeta struct {
	ContentType     string `json:"contentType,omitempty"`
	ContentLength   int64  `json:"contentLength,omitempty"`
	ContentEncoding string `json:"contentEncoding,omitempty"`
}

func (f *LocalFSProvider) Put(ctx context.Context, path string, content BlobContent) error {
	if err := f.writemeta(path, content); err != nil {
		return err
	}
	return f.writedata(path, content)
}

func (f *LocalFSProvider) PutLocation(ctx context.Context, path string) (string, error) {
	return "", modelxerrors.NewUnsupportedError("PutLocation is not supported for local filesystem")
}

func (f *LocalFSProvider) Get(ctx context.Context, path string) (BlobContent, error) {
	meta, err := f.readmeta(path)
	if err != nil {
		return BlobContent{}, err
	}
	stream, err := f.getdata(path)
	if err != nil {
		return BlobContent{}, err
	}
	return BlobContent{
		ContentType:     meta.ContentType,
		ContentLength:   meta.ContentLength,
		ContentEncoding: meta.ContentEncoding,
		Content:         stream,
	}, nil
}

func (f *LocalFSProvider) GetLocation(ctx context.Context, path string) (string, error) {
	return "", modelxerrors.NewUnsupportedError("GetLocation is not supported for local filesystem")
}

func (f *LocalFSProvider) Remove(ctx context.Context, path string, recursive bool) error {
	if recursive {
		return os.RemoveAll(iopath.Join(f.basepath, path))
	}
	return os.Remove(iopath.Join(f.basepath, path))
}

func (f *LocalFSProvider) Exists(ctx context.Context, path string) (bool, error) {
	_, err := os.Stat(iopath.Join(f.basepath, path))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (f *LocalFSProvider) List(ctx context.Context, path string, recursive bool) ([]FsObjectMeta, error) {
	out := []FsObjectMeta{}
	if recursive {
		filepath.WalkDir(iopath.Join(f.basepath, path), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if strings.HasSuffix(path, ".meta") {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			fi, err := d.Info()
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(f.basepath, path)
			if err != nil {
				return err
			}
			out = append(out, FsObjectMeta{
				Name:         rel,
				Size:         fi.Size(),
				LastModified: fi.ModTime(),
			})
			return nil
		})
	} else {
		files, err := os.ReadDir(iopath.Join(f.basepath, path))
		if err != nil {
			return nil, err
		}
		for _, fi := range files {
			if strings.HasSuffix(fi.Name(), ".meta") {
				continue
			}
			if fi.IsDir() {
				continue
			}
			finfo, err := fi.Info()
			if err != nil {
				return nil, err
			}
			out = append(out, FsObjectMeta{
				Name:         fi.Name(),
				Size:         finfo.Size(),
				LastModified: finfo.ModTime(),
			})
		}
	}
	return out, nil
}

func (f *LocalFSProvider) writemeta(path string, content BlobContent) error {
	meta := localFileMeta{
		ContentType:     content.ContentType,
		ContentLength:   content.ContentLength,
		ContentEncoding: content.ContentEncoding,
	}
	jsonData, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	metafile := iopath.Join(f.basepath, path+".meta")
	if err := os.MkdirAll(iopath.Dir(metafile), DefaultDirMode); err != nil {
		return err
	}
	return os.WriteFile(metafile, jsonData, DefaultFileMode)
}

func (f *LocalFSProvider) writedata(path string, content BlobContent) error {
	datafile := iopath.Join(f.basepath, path)
	fi, err := os.OpenFile(datafile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, DefaultFileMode)
	if err != nil {
		return err
	}
	defer fi.Close()
	_, err = io.Copy(fi, content.Content)
	return err
}

func (f *LocalFSProvider) getdata(path string) (io.ReadCloser, error) {
	datafile := iopath.Join(f.basepath, path)
	return os.Open(datafile)
}

func (f *LocalFSProvider) readmeta(path string) (*localFileMeta, error) {
	metafile := iopath.Join(f.basepath, path+".meta")
	raw, err := os.ReadFile(metafile)
	if err != nil {
		return nil, err
	}
	var meta localFileMeta
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}
