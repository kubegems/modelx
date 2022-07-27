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

type Endpoint struct {
	IDs      []string
	Name     string
	Endpoint string
	Handler  http.HandlerFunc
}

// https://github.com/opencontainers/distribution-spec/blob/main/spec.md#endpoints
// | ID     | Method         | API Endpoint                                                 | Success     | Failure           |
// | ------ | -------------- | ------------------------------------------------------------ | ----------- | ----------------- |
// | end-1  | `GET`          | `/v2/`                                                       | `200`       | `404`/`401`       |
// | end-2  | `GET` / `HEAD` | `/v2/<name>/blobs/<digest>`                                  | `200`       | `404`             |
// | end-3  | `GET` / `HEAD` | `/v2/<name>/manifests/<reference>`                           | `200`       | `404`             |
// | end-4a | `POST`         | `/v2/<name>/blobs/uploads/`                                  | `202`       | `404`             |
// | end-4b | `POST`         | `/v2/<name>/blobs/uploads/?digest=<digest>`                  | `201`/`202` | `404`/`400`       |
// | end-5  | `PATCH`        | `/v2/<name>/blobs/uploads/<reference>`                       | `202`       | `404`/`416`       |
// | end-6  | `PUT`          | `/v2/<name>/blobs/uploads/<reference>?digest=<digest>`       | `201`       | `404`/`400`       |
// | end-7  | `PUT`          | `/v2/<name>/manifests/<reference>`                           | `201`       | `404`             |
// | end-8a | `GET`          | `/v2/<name>/tags/list`                                       | `200`       | `404`             |
// | end-8b | `GET`          | `/v2/<name>/tags/list?n=<integer>&last=<integer>`            | `200`       | `404`             |
// | end-9  | `DELETE`       | `/v2/<name>/manifests/<reference>`                           | `202`       | `404`/`400`/`405` |
// | end-10 | `DELETE`       | `/v2/<name>/blobs/<digest>`                                  | `202`       | `404`/`405`       |
// | end-11 | `POST`         | `/v2/<name>/blobs/uploads/?mount=<digest>&from=<other_name>` | `201`       | `404`             |
func (s *Registry) route() http.Handler {
	mux := mux.NewRouter()

	mux = mux.StrictSlash(true)

	// global index
	mux.Methods("GET").Path("/").HandlerFunc(s.GetGlobalIndex)

	// repository
	repository := mux.PathPrefix("/{name:" + NameRegexp + "}").Subrouter()
	// repository/manifests
	manifests := repository.PathPrefix("/manifests").Subrouter()
	manifests.Methods("GET").Path("").HandlerFunc(s.GetIndex)
	manifests.Methods("GET").Path("/{reference:" + ReferenceRegexp + "}").HandlerFunc(s.GetManifest)
	manifests.Methods("PUT").Path("/{reference:" + ReferenceRegexp + "}").HandlerFunc(s.PutManifest)
	manifests.Methods("DELETE").Path("/{reference:" + ReferenceRegexp + "}").HandlerFunc(s.DeleteManifest)

	// repository/blobs
	blobs := repository.PathPrefix("/blobs").Subrouter()
	blobs.Methods("HEAD").Path("/{digest:" + DigestRegexp + "}").HandlerFunc(s.HeadBlob)
	blobs.Methods("GET").Path("/{digest:" + DigestRegexp + "}").HandlerFunc(s.GetBlob)
	blobs.Methods("POST").Path("/{digest:" + DigestRegexp + "}").HandlerFunc(s.PostBlob)
	blobs.Methods("PUT").Path("/{digest:" + DigestRegexp + "}").HandlerFunc(s.PutBlob)

	// repository/uploads
	uploads := blobs.PathPrefix("/uploads").Subrouter()
	uploads.Methods("POST").Path("/{digest:" + DigestRegexp + "}").HandlerFunc(s.PostUpload)
	uploads.Methods("PATCH").Path("/{digest:" + DigestRegexp + "}").HandlerFunc(s.PostUpload)
	return mux
}
