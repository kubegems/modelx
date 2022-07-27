package registry

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"time"
)

type Options struct {
	Listen string
	TLS    *TLSOptions
	S3     *S3Options
	OIDC   *OIDCOptions
}

type OIDCOptions struct {
	Issuer string
}

func DefaultOptions() *Options {
	return &Options{
		Listen: ":8080",
		TLS:    &TLSOptions{},
		S3: &S3Options{
			Buket:         "registry",
			URL:           "https://s3.amazonaws.com",
			AccessKey:     "",
			SecretKey:     "",
			PresignExpire: time.Hour,
			Region:        "",
		},
		OIDC: &OIDCOptions{},
	}
}

type TLSOptions struct {
	CertFile string
	KeyFile  string
	CAFile   string
}

func (t *TLSOptions) ToTLSConfig() (*tls.Config, error) {
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

type S3Options struct {
	URL           string        `json:"url,omitempty"`
	Region        string        `json:"region,omitempty"`
	Buket         string        `json:"buket,omitempty"`
	AccessKey     string        `json:"accessKey,omitempty"`
	SecretKey     string        `json:"secretKey,omitempty"`
	PresignExpire time.Duration `json:"presignExpire,omitempty"`
}
