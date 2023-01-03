package pkg

import (
	"os"

	"sigs.k8s.io/yaml"
)

type Target struct {
	ObjectType string   `json:"type"`
	Namespace  string   `json:"namespace"`
	Name       string   `json:"name"`
	Ports      []string `json:"ports"`
}

func (t Target) String() string {
	//TODO:
	return t.Namespace + "/" + t.Name
}

type Config struct {
	Targets []Target `json:"targets"`
}

func LoadConfig(filepath string) (*Config, error) {
	b, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	err = yaml.Unmarshal(b, cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
