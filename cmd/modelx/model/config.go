package model

const (
	ModelConfigFileName = "modelx.yaml"
	ReadmeFileName      = "README.md"
)

type ModelConfig struct {
	Description string            `json:"description"`
	FrameWork   string            `json:"framework"`
	Task        string            `json:"task"`
	Tags        []string          `json:"tags"`
	Resources   map[string]any    `json:"resources"`
	Mantainers  []string          `json:"maintainers"`
	Annotations map[string]string `json:"annotations,omitempty"`
	ModelFiles  []string          `json:"modelFiles"`
	Config      any               `json:"config"`
}
