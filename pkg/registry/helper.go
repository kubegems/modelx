package registry

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-logr/logr"
	apierr "kubegems.io/modelx/pkg/errors"
)

const MediaTypeModelIndexJson = "application/vnd.modelx.model.index.v1.json"

const MaxBytesRead = int64(1 << 20) // 1MB

// MaxBytesReadHandler returns a Handler that runs h with its ResponseWriter and Request.Body wrapped by a MaxBytesReader.
func MaxBytesReadHandler(h http.HandlerFunc, n int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r2 := *r
		r2.Body = http.MaxBytesReader(w, r.Body, n)
		h.ServeHTTP(w, &r2)
	}
}

func ResponseError(w http.ResponseWriter, err error) {
	info := apierr.ErrorInfo{}
	if !errors.As(err, &info) {
		info = apierr.ErrorInfo{
			HttpStatus: http.StatusBadRequest,
			Code:       apierr.ErrCodeUnknow,
			Message:    err.Error(),
			Detail:     err.Error(),
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(info.HttpStatus)
	json.NewEncoder(w).Encode(info)
}

func ResponseOK(w http.ResponseWriter, data any) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}

type contextUsernameKey struct{}

func UsernameFromContext(ctx context.Context) string {
	if username, ok := ctx.Value(contextUsernameKey{}).(string); ok {
		return username
	}
	return ""
}

func NewUsernameContext(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, contextUsernameKey{}, username)
}

func NewOIDCAuthFilter(ctx context.Context, issuer string, next http.Handler) http.Handler {
	ctx = oidc.InsecureIssuerURLContext(ctx, issuer)
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		log.Fatal(err)
	}
	verifier := provider.Verifier(&oidc.Config{
		SkipClientIDCheck: true,
		SkipIssuerCheck:   true,
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headerAuthorzation := r.Header.Get("Authorization")
		token := strings.TrimPrefix(headerAuthorzation, "Bearer ")
		if token == "" {
			queries := r.URL.Query()
			for _, k := range []string{"token", "access_token"} {
				if token = queries.Get(k); token != "" {
					break
				}
			}
		}
		if len(token) == 0 {
			ResponseError(w, apierr.NewUnauthorizedError("missing access token"))
			return
		}
		idtoken, err := verifier.Verify(r.Context(), token)
		if err != nil {
			ResponseError(w, apierr.NewUnauthorizedError("invalid access token"))
			return
		}
		r.WithContext(NewUsernameContext(r.Context(), idtoken.Subject))
		next.ServeHTTP(w, r)
	})
}

func LoggingFilter(logger logr.Logger, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		from := time.Now()

		h.ServeHTTP(w, r)

		cost := time.Since(from)
		logger.Info("http",
			"method", r.Method,
			"path", r.RequestURI,
			"cost", cost,
			"proto", r.Proto,
			"ua", r.UserAgent(),
		)
	})
}
