package types

import (
	"math/big"
	"net/url"
	"path"
	"strconv"
	"strings"
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
		Path:   path.Join(s.Path, u.ProjectAddress, u.AssetID.String()),
		RawQuery: url.Values{
			"access-grant": {u.AccessGrant},
			"username":     {u.Username},
			"file":         {u.File},
		}.Encode(),
	}).String(), nil
}

func ParseVaultURL(in string) (*VaultBlobMeta, error) {
	u, err := url.Parse(in)
	if err != nil {
		return nil, err
	}
	ret := &VaultBlobMeta{
		ServiceUrl:  u.Host,
		AccessGrant: u.Query().Get("access-grant"),
		Username:    u.Query().Get("username"),
		File:        u.Query().Get("file"),
	}
	splites := strings.Split(u.Path, "/")
	// splites: []string len: 3, cap: 3, ["","0x1234567890abcdef","1"]
	if len(splites) > 1 {
		ret.ProjectAddress = splites[1]
	}
	if len(splites) > 2 {
		parsedint, err := strconv.Atoi(splites[2])
		if err != nil {
			return nil, err
		}
		ret.AssetID = big.NewInt(int64(parsedint))
	}
	return ret, nil
}
