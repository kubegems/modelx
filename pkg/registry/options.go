package registry

type Options struct {
	Listen         string
	TLS            *TLSOptions
	S3             *S3Options
	Local          *LocalFSOptions
	EnableRedirect bool
	OIDC           *OIDCOptions
	Vault          *VaultOptions
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
		Vault:          NewDefaultVaultOptions(),
	}
}

type TLSOptions struct {
	CertFile string
	KeyFile  string
	CAFile   string
}
