package types

import (
	"os"
	"strings"
	"time"

	"github.com/opencontainers/go-digest"
)

const (
	AnnotationFileMode = "filemode"
)

const (
	BlobLocationPurposeUpload   string = "upload"
	BlobLocationPurposeDownload string = "download"
)

type BlobLocation struct {
	Provider   string     `json:"provider,omitempty"`
	Purpose    string     `json:"purpose,omitempty"`
	Properties Properties `json:"properties,omitempty"`
}

type Properties map[string]any

type Descriptor struct {
	Name        string        `json:"name"`
	MediaType   string        `json:"mediaType,omitempty"`
	Digest      digest.Digest `json:"digest,omitempty"`
	Size        int64         `json:"size,omitempty"`
	Mode        os.FileMode   `json:"mode,omitempty"`
	URLs        []string      `json:"urls,omitempty"`
	Modified    time.Time     `json:"modified,omitempty"`
	Annotations Annotations   `json:"annotations,omitempty"`
}

type Annotations map[string]string

func (a Annotations) String() string {
	var result []string
	for k, v := range a {
		result = append(result, k+"="+v)
	}
	return strings.Join(result, ",")
}

func SortDescriptorName(a, b Descriptor) bool {
	return strings.Compare(a.Name, b.Name) < 0
}

type Index struct {
	SchemaVersion int               `json:"schemaVersion"`
	MediaType     string            `json:"mediaType,omitempty"`
	Manifests     []Descriptor      `json:"manifests"`
	Annotations   map[string]string `json:"annotations,omitempty"`
}

type Manifest struct {
	SchemaVersion int               `json:"schemaVersion"`
	MediaType     string            `json:"mediaType,omitempty"`
	Config        Descriptor        `json:"config"`
	Blobs         []Descriptor      `json:"blobs"`
	Annotations   map[string]string `json:"annotations,omitempty"`
}
