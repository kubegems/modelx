package registry

import (
	"context"
	"encoding/json"
	"io"
	"os"
	iopath "path"
	"path/filepath"
	"strings"
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
	ContentType   string `json:"contentType,omitempty"`
	ContentLength int64  `json:"contentLength,omitempty"`
}

func (f *LocalFSProvider) Put(ctx context.Context, path string, content BlobContent) error {
	if err := f.writemeta(path, content); err != nil {
		return err
	}
	return f.writedata(path, content)
}

func (f *LocalFSProvider) Get(ctx context.Context, path string) (*BlobContent, error) {
	meta, err := f.readmeta(path)
	if err != nil {
		return nil, err
	}
	stream, err := f.getdata(path)
	if err != nil {
		return nil, err
	}
	return &BlobContent{
		ContentType: meta.ContentType,

		Content: stream,
	}, nil
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

func (f *LocalFSProvider) Stat(ctx context.Context, path string) (FsObjectMeta, error) {
	fi, err := os.Stat(iopath.Join(f.basepath, path))
	if err != nil {
		return FsObjectMeta{}, err
	}
	meta, _ := f.readmeta(path)
	return FsObjectMeta{
		Name:         path,
		Size:         fi.Size(),
		LastModified: fi.ModTime(),
		ContentType:  meta.ContentType,
	}, nil
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
		ContentType:   content.ContentType,
		ContentLength: content.ContentLength,
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
	fi, err := os.Stat(iopath.Join(f.basepath, path))
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		return nil, os.ErrNotExist
	}
	metafile := iopath.Join(f.basepath, path+".meta")
	raw, err := os.ReadFile(metafile)
	if err != nil {
		return nil, err
	}
	var meta localFileMeta
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil, err
	}
	meta.ContentLength = fi.Size()
	return &meta, nil
}
