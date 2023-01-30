package client

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mholt/archiver/v4"
	"github.com/opencontainers/go-digest"
	"golang.org/x/sync/errgroup"
	"kubegems.io/modelx/pkg/types"
)

type DescriptorWithContent struct {
	types.Descriptor
	Content GetContentFunc
}

func ParseDir(ctx context.Context, dir string) (map[string]DescriptorWithContent, error) {
	files := sync.Map{}

	ds, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	wg := errgroup.Group{}
	wg.SetLimit(PushConcurrency)
	for _, d := range ds {
		d := d
		wg.Go(func() error {
			if strings.HasPrefix(d.Name(), ".") {
				return nil
			}
			filename := filepath.Join(dir, d.Name())
			fi, err := d.Info()
			if err != nil {
				return err
			}

			if d.IsDir() {
				tgzfile := filepath.Join(dir, ".modelx", d.Name()+".tar.gz")
				digest, err := TGZ(ctx, filename, tgzfile)
				if err != nil {
					return err
				}
				tgzfi, err := os.Stat(tgzfile)
				if err != nil {
					return err
				}
				files.Store(d.Name(), DescriptorWithContent{
					Descriptor: types.Descriptor{
						Name:      d.Name(),
						MediaType: MediaTypeModelDirectoryTarGz,
						Digest:    digest,
						Size:      tgzfi.Size(),
						Modified:  fi.ModTime(),
						Mode:      fi.Mode(),
					},
					Content: func() (io.ReadSeekCloser, error) {
						return os.Open(tgzfile)
					},
				})
				return nil
			}

			getReader := func() (io.ReadSeekCloser, error) {
				return os.Open(filename)
			}

			f, err := getReader()
			if err != nil {
				return err
			}
			defer f.Close()

			desc, err := digest.FromReader(f)
			if err != nil {
				return err
			}

			files.Store(d.Name(), DescriptorWithContent{
				Descriptor: types.Descriptor{
					Name:      d.Name(),
					MediaType: MediaTypeModelFile,
					Digest:    desc,
					Size:      fi.Size(),
					Modified:  fi.ModTime(),
					Mode:      fi.Mode(),
				},
				Content: getReader,
			})
			return nil
		})
	}
	if err := wg.Wait(); err != nil {
		return nil, err
	}

	bodymap := map[string]DescriptorWithContent{}
	files.Range(func(key, value any) bool {
		bodymap[key.(string)] = value.(DescriptorWithContent)
		return true
	})
	return bodymap, nil
}

var tgz = archiver.CompressedArchive{
	Archival:    archiver.Tar{},
	Compression: archiver.Gz{},
}

func TGZ(ctx context.Context, dir string, intofile string) (digest.Digest, error) {
	files, err := archiver.FilesFromDisk(
		&archiver.FromDiskOptions{ClearAttributes: true},
		map[string]string{dir + string(os.PathSeparator): ""},
	)
	if err != nil {
		return "", err
	}

	writers := []io.Writer{}
	if intofile != "" {
		if err := os.MkdirAll(filepath.Dir(intofile), 0o755); err != nil {
			return "", err
		}
		f, err := os.Create(intofile)
		if err != nil {
			return "", err
		}
		defer f.Close()

		writers = append(writers, f)
	}
	d := digest.Canonical.Digester()
	writers = append(writers, d.Hash())

	if err := tgz.Archive(ctx, io.MultiWriter(writers...), files); err != nil {
		return "", err
	}
	return d.Digest(), nil
}

func UnTGZFile(ctx context.Context, intodir string, file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := UnTGZ(ctx, intodir, f); err != nil {
		return err
	}
	return nil
}

func UnTGZ(ctx context.Context, intodir string, readercloser io.Reader) error {
	return tgz.Extract(ctx, readercloser, nil, func(ctx context.Context, f archiver.File) error {
		nameinlocal := filepath.Join(intodir, f.NameInArchive)
		if f.IsDir() {
			return os.MkdirAll(nameinlocal, f.Mode())
		}
		srcfile, err := f.Open()
		if err != nil {
			return err
		}
		defer srcfile.Close()

		intofile, err := os.OpenFile(nameinlocal, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		defer intofile.Close()

		_, err = io.Copy(intofile, srcfile)
		if err != nil {
			return err
		}
		return nil
	})
}
