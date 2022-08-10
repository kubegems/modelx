package registry

import (
	"time"
)

type Options struct {
	Listen         string
	TLS            *TLSOptions
	S3             *S3Options
	EnableRedirect bool
	OIDC           *OIDCOptions
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

type S3Options struct {
	URL           string        `json:"url,omitempty"`
	Region        string        `json:"region,omitempty"`
	Buket         string        `json:"buket,omitempty"`
	AccessKey     string        `json:"accessKey,omitempty"`
	SecretKey     string        `json:"secretKey,omitempty"`
	PresignExpire time.Duration `json:"presignExpire,omitempty"`
}
