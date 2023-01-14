package pkg

import (
	"fmt"
	"os"
	"strings"

	"sigs.k8s.io/yaml"
)

type Target struct {
	ObjectType string   `json:"type"`
	Namespace  string   `json:"namespace"`
	Name       string   `json:"name"`
	Ports      []string `json:"ports"`
}

func (t Target) String() string {
	return fmt.Sprintf("%s:%s/%s(%s)", t.ObjectType, t.Namespace, t.Name, strings.Join(t.Ports, ","))
}

type Manifest struct {
	Targets []Target `json:"targets"`
}

func LoadManifest(filepath string) (*Manifest, error) {
	b, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	cfg := &Manifest{}
	err = yaml.Unmarshal(b, cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
