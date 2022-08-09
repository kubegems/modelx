package registry

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	autherrors "github.com/go-oauth2/oauth2/v4/errors"
	"github.com/go-oauth2/oauth2/v4/manage"
	"github.com/go-oauth2/oauth2/v4/server"
	"github.com/golang-jwt/jwt"
	"kubegems.io/modelx/pkg/errors"
)

var (
	srv      *server.Server
	oauthreq *http.Request
	tokenMap sync.Map
)

func Init(opts *OauthOptions) {
	manager := manage.NewDefaultManager()
	srv = server.NewServer(server.NewConfig(), manager)
	oauthreq, _ = http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s%s?grant_type=client_credentials&scope=validate", opts.Server, opts.ValidatePath),
		nil,
	)
	go func() {
		for range time.NewTicker(10 * time.Second).C {
			tokenMap.Range(func(key, value any) bool {
				exp := value.(time.Time)
				if time.Now().After(exp) {
					tokenMap.Delete(key)
				}
				log.Printf("token: %v, exp: %s, remain: %s", key, exp.String(), exp.Sub(time.Now()).String())
				return false
			})
		}
	}()
}

func (s *Registry) Oauth(w http.ResponseWriter, r *http.Request) {
	token, ok := srv.BearerAuth(r)
	if !ok {
		ResponseError(w, errors.NewAuthFailedError(autherrors.ErrInvalidAccessToken))
		return
	}
	claims := jwt.StandardClaims{}
	_, _, err := new(jwt.Parser).ParseUnverified(token, &claims)
	if err != nil {
		ResponseError(w, errors.NewAuthFailedError(err))
		return
	}

	newreq := oauthreq.WithContext(r.Context())
	newreq.Header = http.Header{
		"Authorization": []string{"Bearer " + token},
	}
	resp, err := http.DefaultClient.Do(newreq)
	if err != nil {
		ResponseError(w, errors.NewAuthFailedError(err))
		return
	}
	msg, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		ResponseError(w, errors.NewAuthFailedError(fmt.Errorf(string(msg))))
		return
	}

	exp := time.Unix(claims.ExpiresAt, 0)
	tokenMap.LoadOrStore(token, exp)
	log.Printf("user %s login, token expiresAt: %s", claims.Audience, exp.String())
	ResponseOK(w, string(msg))
}

func NewOauthFilter(ctx context.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth" {
			next.ServeHTTP(w, r)
			return
		}
		token, ok := srv.BearerAuth(r)
		if !ok {
			ResponseError(w, errors.NewAuthFailedError(autherrors.ErrInvalidAccessToken))
			return
		}
		value, ok := tokenMap.Load(token)
		if !ok {
			ResponseError(w, errors.NewAuthFailedError(fmt.Errorf("you are not login or token has expired")))
			return
		}
		exp := value.(time.Time)
		if time.Now().After(exp) {
			ResponseError(w, errors.NewAuthFailedError(fmt.Errorf("you are not login or token has expired")))
			tokenMap.Delete(token)
			return
		}
		next.ServeHTTP(w, r)
	})
}
