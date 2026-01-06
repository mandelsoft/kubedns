package config

import (
	"os"

	"k8s.io/client-go/rest"
)

type InClusterConfig struct{}

func NewInClusterConfig() Rule {
	return InClusterConfig{}
}

func (r InClusterConfig) GetConfig(opts *ConfigOptions) (*Config, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		if err == os.ErrNotExist || err == rest.ErrNotInCluster {
			return nil, nil
		}
		return nil, err
	}
	return &Config{RestConfig: cfg, Identity: opts.Idenitity, Context: opts.CurrentContext}, nil
}
