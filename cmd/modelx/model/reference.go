package model

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"kubegems.io/modelx/cmd/modelx/repo"
	"kubegems.io/modelx/pkg/client"
)

const ModelxAuthEnv = "MODELX_AUTH"

type Reference struct {
	Registry      string
	Repository    string
	Version       string
	Authorization string
}

func (r Reference) String() string {
	if r.Version == "" {
		return fmt.Sprintf("%s/%s", r.Registry, r.Repository)
	}
	return fmt.Sprintf("%s/%s@%s", r.Registry, r.Repository, r.Version)
}

func (r Reference) Client() *client.Client {
	return client.NewClient(r.Registry, r.Authorization)
}

func ParseReference(raw string) (Reference, error) {
	auth := os.Getenv(ModelxAuthEnv)
	if !strings.Contains(raw, "://") {
		splits := strings.SplitN(raw, repo.SplitorRepo, 2)
		details, err := repo.DefaultRepoManager.Get(splits[0])
		if err != nil {
			return Reference{}, err
		}
		if auth == "" {
			auth = "Bearer " + details.Token
		}
		if len(splits) == 2 {
			raw = details.URL + "/" + splits[1]
		} else {
			raw = details.URL
		}
	}

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
	if token := u.Query().Get("token"); token != "" {
		auth = "Bearer " + token
	}
	repository, version := "", ""
	splits := strings.SplitN(u.Path, repo.SplitorVersion, 2)
	if len(splits) != 2 || splits[1] == "" {
		version = ""
	} else {
		version = splits[1]
	}
	if sp0 := splits[0]; sp0 != "" {
		repository = sp0[1:]
	}

	if repository != "" && !strings.Contains(repository, "/") {
		repository = "library/" + repository
	}

	ref := Reference{
		Registry:      u.Scheme + "://" + u.Host,
		Repository:    repository,
		Version:       version,
		Authorization: auth,
	}
	return ref, nil
}
