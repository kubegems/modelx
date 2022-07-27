package registry

import (
	"net/http"
	"sort"
	"strings"
)

func MethodHandler(h map[string]http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if handler, ok := h[r.Method]; ok {
			handler.ServeHTTP(w, r)
		} else {
			allow := []string{}
			for k := range h {
				allow = append(allow, k)
			}
			sort.Strings(allow)
			w.Header().Set("Allow", strings.Join(allow, ", "))
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		}
	}
}

// MaxBytesReadHandler returns a Handler that runs h with its ResponseWriter and Request.Body wrapped by a MaxBytesReader.
func MaxBytesReadHandler(h http.HandlerFunc, n int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r2 := *r
		r2.Body = http.MaxBytesReader(w, r.Body, n)
		h.ServeHTTP(w, &r2)
	}
}
