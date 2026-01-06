package config

import (
	"os"

	"github.com/mandelsoft/goutils/general"
)

type EnvironmentVariable struct {
	Name string
}

func NewEnvironmentVariable(name ...string) *EnvironmentVariable {
	return &EnvironmentVariable{general.OptionalDefaulted("KUBECONFIG", name...)}
}

func (r *EnvironmentVariable) GetConfig(opts *ConfigOptions) (*Config, error) {
	v := os.Getenv(r.Name)
	if v == "" {
		return nil, nil
	}
	cfg, err := TryKubeconfigFile(v, opts)
	if err != nil {
		return nil, err
	}
	return cfg, err
}
