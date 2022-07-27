package client

import (
	"fmt"
	"net/url"
	"strings"
)

type Reference struct {
	Registry   string
	Repository string
	Version    string
}

func (r Reference) String() string {
	if r.Version == "" {
		return fmt.Sprintf("%s/%s", r.Registry, r.Repository)
	}
	return fmt.Sprintf("%s/%s@%s", r.Registry, r.Repository, r.Version)
}

func ParseReference(raw string) (Reference, error) {
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		raw = "https://" + raw
	}
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return Reference{}, fmt.Errorf("invalid reference: %s", err)
	}
	if u.Host == "" {
		return Reference{}, fmt.Errorf("invalid reference: missing host")
	}
	repository, version := "", ""
	splits := strings.SplitN(u.Path, "@", 2)
	if len(splits) != 2 || splits[1] == "" {
		version = ""
	} else {
		version = splits[1]
	}
	if sp0 := splits[0]; sp0 != "" {
		repository = sp0[1:]
	}
	ref := Reference{
		Registry:   u.Scheme + "://" + u.Host,
		Repository: repository,
		Version:    version,
	}
	return ref, nil
}
