package vault

import (
	"math/big"
	"net/url"
	"strconv"
)

type VaultBlobMeta struct {
	ServiceUrl     string
	ProjectAddress string
	AccessGrant    string
	AssetID        *big.Int
	Username       string
	File           string
}

func (u VaultBlobMeta) ToURL() (string, error) {
	schema := "idoe"
	s, err := url.Parse(u.ServiceUrl)
	if err != nil {
		return "", err
	}
	if s.Scheme == "https" {
		schema = "idoes"
	}
	return (&url.URL{
		Scheme: schema,
		Host:   s.Host,
		Path:   s.Path,
		RawQuery: url.Values{
			"project-address": {u.ProjectAddress},
			"asset-id":        {u.AssetID.String()},
			"access-grant":    {u.AccessGrant},
			"username":        {u.Username},
			"file":            {u.File},
		}.Encode(),
	}).String(), nil
}

func ParseVaultURL(in string) (*VaultBlobMeta, error) {
	u, err := url.Parse(in)
	if err != nil {
		return nil, err
	}
	schema := "http"
	if u.Scheme == "idoes" {
		schema = "https"
	}
	queries := u.Query()
	parsedint, err := strconv.Atoi(queries.Get("asset-id"))
	if err != nil {
		return nil, err
	}
	ret := &VaultBlobMeta{
		ServiceUrl:     (&url.URL{Scheme: schema, Host: u.Host, Path: u.Path}).String(),
		AccessGrant:    queries.Get("access-grant"),
		Username:       queries.Get("username"),
		File:           queries.Get("file"),
		ProjectAddress: queries.Get("project-address"),
		AssetID:        big.NewInt(int64(parsedint)),
	}
	return ret, nil
}
