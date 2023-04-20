package registry

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
		S3:     NewDefaultS3Options(),
		OIDC:   &OIDCOptions{},
	}
}

type TLSOptions struct {
	CertFile string
	KeyFile  string
	CAFile   string
}
