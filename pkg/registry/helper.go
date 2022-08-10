package registry

import (
	"net/http"
)

const MaxBytesRead = int64(1 << 20) // 1MB

// MaxBytesReadHandler returns a Handler that runs h with its ResponseWriter and Request.Body wrapped by a MaxBytesReader.
func MaxBytesReadHandler(h http.HandlerFunc, n int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r2 := *r
		r2.Body = http.MaxBytesReader(w, r.Body, n)
		h.ServeHTTP(w, &r2)
	}
}
