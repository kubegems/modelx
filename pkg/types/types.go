package types

import (
	"strings"
	"time"

	"github.com/opencontainers/go-digest"
)

const (
	MediaTypeModelIndexJson    = "application/vnd.modelx.model.index.v1.json"
	MediaTypeModelManifestYaml = "application/vnd.modelx.model.manifest.v1+yaml"
	MediaTypeModelConfigYaml   = "application/vnd.modelx.model.config.v1+yaml"
	MediaTypeModelFile         = "application/vnd.modelx.model.file.v1"

	RegistryIndexFileName = "index.json"
)

type Descriptor struct {
	Name        string            `json:"name"`
	MediaType   string            `json:"mediaType,omitempty"`
	Digest      digest.Digest     `json:"digest,omitempty"`
	Size        int64             `json:"size,omitempty"`
	URLs        []string          `json:"urls,omitempty"`
	Modified    time.Time         `json:"modified,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
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
