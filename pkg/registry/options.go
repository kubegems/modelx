package registry

type Options struct {
	Listen         string
	TLS            *TLSOptions
	S3             *S3Options
	Local          *LocalFSOptions
	EnableRedirect bool
	OIDC           *OIDCOptions
}

type OIDCOptions struct {
	Issuer string
}

func DefaultOptions() *Options {
	return &Options{
		Listen:         ":8080",
		TLS:            &TLSOptions{},
		S3:             NewDefaultS3Options(),
		OIDC:           &OIDCOptions{},
		Local:          NewDefaultLocalFSOptions(),
		EnableRedirect: false, // default to false
	}
}

type TLSOptions struct {
	CertFile string
	KeyFile  string
	CAFile   string
}
