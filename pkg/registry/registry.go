package registry

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/opencontainers/go-digest"
	"kubegems.io/modelx/pkg/errors"
	"kubegems.io/modelx/pkg/types"
)

const ManifestContentLimit = int64(4 * 1024) // 4mb

type Options struct {
	Listen string
	TLS    *TLS
	S3     *S3
}

func DefaultOptions() *Options {
	return &Options{
		Listen: ":8080",
		TLS:    &TLS{},
		S3: &S3{
			Buket:         "registry",
			URL:           "https://s3.amazonaws.com",
			AccessKey:     "",
			SecretKey:     "",
			PresignExpire: time.Hour,
			Region:        "",
		},
	}
}

type TLS struct {
	CertFile string
	KeyFile  string
	CAFile   string
}

func (t *TLS) ToTLSConfig() (*tls.Config, error) {
	cafile, certfile, keyfile := t.CAFile, t.CertFile, t.KeyFile

	pool, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}
	config := &tls.Config{ClientCAs: pool}
	if cafile != "" {
		capem, err := ioutil.ReadFile(cafile)
		if err != nil {
			return nil, err
		}
		config.ClientCAs.AppendCertsFromPEM(capem)
	}
	certificate, err := tls.LoadX509KeyPair(certfile, keyfile)
	if err != nil {
		return nil, err
	}
	config.Certificates = append(config.Certificates, certificate)
	return config, nil
}

type S3 struct {
	URL           string        `json:"url,omitempty"`
	Region        string        `json:"region,omitempty"`
	Buket         string        `json:"buket,omitempty"`
	AccessKey     string        `json:"accessKey,omitempty"`
	SecretKey     string        `json:"secretKey,omitempty"`
	PresignExpire time.Duration `json:"presignExpire,omitempty"`
}

func Run(ctx context.Context, opts *Options) error {
	registry, err := NewRegistry(ctx, opts)
	if err != nil {
		return err
	}

	loggedRouter := handlers.CombinedLoggingHandler(os.Stdout, registry.route())

	server := http.Server{
		Addr:    opts.Listen,
		Handler: loggedRouter,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}
	go func() {
		<-ctx.Done()
		server.Shutdown(ctx)
	}()
	if opts.TLS.CertFile != "" && opts.TLS.KeyFile != "" {
		log.Printf("registry listening on https: %s", opts.Listen)
		return server.ListenAndServeTLS(opts.TLS.CertFile, opts.TLS.KeyFile)
	} else {
		log.Printf("registry listening on http %s", opts.Listen)
		return server.ListenAndServe()
	}
}

func NewRegistry(ctx context.Context, opt *Options) (*Registry, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(opt.S3.AccessKey, opt.S3.SecretKey, ""),
		),
		config.WithRegion(opt.S3.Region),
		config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{URL: opt.S3.URL}, nil
				},
			),
		),
	)
	if err != nil {
		return nil, err
	}
	s3cli := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})
	store := &RegistryStore{
		Storage: &S3StorageProvider{
			Bucket:  opt.S3.Buket,
			Client:  s3cli,
			PreSign: s3.NewPresignClient(s3cli),
			Expire:  opt.S3.PresignExpire,
			Prefix:  "registry",
		},
	}
	return &Registry{Manifest: store}, nil
}

type Registry struct {
	Manifest *RegistryStore
}

func (s *Registry) HeadManifest(w http.ResponseWriter, r *http.Request) {
	name, reference := GetRepositoryReference(r)
	exist, err := s.Manifest.Exists(r.Context(), name, reference)
	if err != nil {
		ResponseError(w, err)
		return
	}
	if exist {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func (s *Registry) GetGlobalIndex(w http.ResponseWriter, r *http.Request) {
	index, err := s.Manifest.GetGlobalIndex(r.Context(), r.URL.Query().Get("search"))
	if err != nil {
		ResponseError(w, err)
		return
	}
	ResponseOK(w, index)
}

func (s *Registry) GetIndex(w http.ResponseWriter, r *http.Request) {
	name, _ := GetRepositoryReference(r)
	index, err := s.Manifest.GetIndex(r.Context(), name, r.URL.Query().Get("search"))
	if err != nil {
		if IsStorageNotFound(err) {
			err = errors.NewIndexUnknownError(name)
		}
		ResponseError(w, err)
		return
	}
	ResponseOK(w, index)
}

func (s *Registry) GetManifest(w http.ResponseWriter, r *http.Request) {
	name, reference := GetRepositoryReference(r)
	manifest, err := s.Manifest.GetManifest(r.Context(), name, reference)
	if err != nil {
		ResponseError(w, err)
		return
	}
	ResponseOK(w, manifest)
}

func (s *Registry) PutManifest(w http.ResponseWriter, r *http.Request) {
	name, reference := GetRepositoryReference(r)
	var manifest types.Manifest
	if err := json.NewDecoder(r.Body).Decode(&manifest); err != nil {
		ResponseError(w, errors.NewManifestInvalidError(err))
		return
	}
	contenttype := r.Header.Get("Content-Type")
	if err := s.Manifest.PutManifest(r.Context(), name, reference, contenttype, manifest); err != nil {
		ResponseError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Registry) DeleteManifest(w http.ResponseWriter, r *http.Request) {
	name, reference := GetRepositoryReference(r)
	if err := s.Manifest.DeleteManifest(r.Context(), name, reference); err != nil {
		ResponseError(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Registry) PostUpload(w http.ResponseWriter, r *http.Request) {
}

func GetRepositoryReference(r *http.Request) (string, string) {
	vars := mux.Vars(r)
	return vars["name"], vars["reference"]
}

func (s *Registry) HeadBlob(w http.ResponseWriter, r *http.Request) {
	BlobDigestFun(w, r, func(ctx context.Context, repository string, digest digest.Digest) {
		ok, err := s.Manifest.ExistsBlob(r.Context(), repository, digest)
		if err != nil {
			ResponseError(w, err)
			return
		}
		if ok {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

// 如果客户端 包含 contentLength 则直接上传
// 如果客户端 不包含 contentLength 则返回一个 Location 后续上传至该地址
func (s *Registry) PostBlob(w http.ResponseWriter, r *http.Request) {
	s.PutBlob(w, r)
}

func (s *Registry) PutBlob(w http.ResponseWriter, r *http.Request) {
	repository, _ := GetRepositoryReference(r)
	desc, err := ParseDescriptor(r)
	if err != nil {
		ResponseError(w, err)
		return
	}
	if err := s.Manifest.PutBlob(r.Context(), repository, *desc, r.Body); err != nil {
		ResponseError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Registry) GetBlob(w http.ResponseWriter, r *http.Request) {
	BlobDigestFun(w, r, func(ctx context.Context, repository string, digest digest.Digest) {
		location, err := s.Manifest.GetBlobURL(r.Context(), repository, digest)
		if err != nil {
			if !errors.IsErrCode(err, errors.ErrCodeUnsupported) {
				ResponseError(w, err)
				return
			}
			rc, err := s.Manifest.GetBlob(r.Context(), repository, digest)
			if err != nil {
				ResponseError(w, err)
				return
			}
			w.Header().Set("Content-Length", strconv.Itoa(int(rc.ContentLength)))
			w.Header().Set("Content-Type", rc.ContentType)
			w.WriteHeader(http.StatusOK)
			io.Copy(w, rc)
			return
		}
		w.Header().Add("Location", location)
		w.WriteHeader(http.StatusFound)
	})
}

func BlobDigestFun(w http.ResponseWriter, r *http.Request, fun func(ctx context.Context, repository string, digest digest.Digest)) {
	name, _ := GetRepositoryReference(r)
	digeststr := mux.Vars(r)["digest"]
	digest, err := digest.Parse(digeststr)
	if err != nil {
		ResponseError(w, errors.NewDigestInvalidError(digeststr))
		return
	}
	fun(r.Context(), name, digest)
}

func ParseDescriptor(r *http.Request) (*types.Descriptor, error) {
	digeststr := mux.Vars(r)["digest"]
	digest, err := digest.Parse(digeststr)
	if err != nil {
		return nil, errors.NewDigestInvalidError(digeststr)
	}
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		return nil, errors.NewContentTypeInvalidError("empty")
	}
	descriptor := &types.Descriptor{
		Digest:    digest,
		MediaType: contentType,
	}
	return descriptor, nil
}

func ParseAndCheckContentRange(header http.Header) (int64, int64, error) {
	contentRange, contentLength := header.Get("Content-Range"), header.Get("Content-Length")
	ranges := strings.Split(contentRange, "-")
	if len(ranges) != 2 {
		return -1, -1, errors.NewContentRangeInvalidError("invalid format")
	}
	start, err := strconv.ParseInt(ranges[0], 10, 64)
	if err != nil {
		return -1, -1, errors.NewContentRangeInvalidError("invalid start")
	}
	end, err := strconv.ParseInt(ranges[1], 10, 64)
	if err != nil {
		return -1, -1, errors.NewContentRangeInvalidError("invalid end")
	}
	if start > end {
		return -1, -1, errors.NewContentRangeInvalidError("start > end")
	}
	contentLen, err := strconv.ParseInt(contentLength, 10, 64)
	if err != nil {
		return -1, -1, errors.NewContentRangeInvalidError("invalid content length")
	}
	if contentLen != (end-start)+1 {
		return -1, -1, errors.NewContentRangeInvalidError("content length != (end-start)+1")
	}
	return start, end, nil
}
