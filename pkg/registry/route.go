package registry

import (
	"net/http"

	"github.com/gorilla/mux"
)

const (
	NameRegexp      = `[a-z0-9]+(?:[._-][a-z0-9]+)*/(?:[a-z0-9]+(?:[._-][a-z0-9]+)*)`
	ReferenceRegexp = `[a-zA-Z0-9_][a-zA-Z0-9._-]{0,127}`
	DigestRegexp    = `[A-Za-z][A-Za-z0-9]*(?:[-_+.][A-Za-z][A-Za-z0-9]*)*[:][[:xdigit:]]{32,}`
)

func (s *Registry) route() http.Handler {
	mux := mux.NewRouter()
	mux = mux.StrictSlash(true)
	// healthy
	mux.Methods("GET").Path("/healthz").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
		w.WriteHeader(http.StatusOK)
	})
	// global index
	mux.Methods("GET").Path("/").HandlerFunc(s.GetGlobalIndex)
	// repository
	repository := mux.PathPrefix("/{name:" + NameRegexp + "}").Subrouter()
	// index
	repository.Methods("GET").Path("/index").HandlerFunc(s.GetIndex)
	repository.Methods("DELETE").Path("/index").HandlerFunc(s.DeleteIndex)
	// repository/manifests
	manifests := repository.PathPrefix("/manifests").Subrouter()
	manifests.Methods("GET").Path("/{reference:" + ReferenceRegexp + "}").HandlerFunc(s.GetManifest)
	manifests.Methods("PUT").Path("/{reference:" + ReferenceRegexp + "}").HandlerFunc(MaxBytesReadHandler(s.PutManifest, MaxBytesRead))
	manifests.Methods("DELETE").Path("/{reference:" + ReferenceRegexp + "}").HandlerFunc(s.DeleteManifest)

	// repository/blobs
	blobs := repository.PathPrefix("/blobs").Subrouter()
	blobs.Methods("HEAD").Path("/{digest:" + DigestRegexp + "}").HandlerFunc(s.HeadBlob)
	blobs.Methods("GET").Path("/{digest:" + DigestRegexp + "}").HandlerFunc(s.GetBlob)
	blobs.Methods("PUT").Path("/{digest:" + DigestRegexp + "}").HandlerFunc(s.PutBlob)

	return mux
}
