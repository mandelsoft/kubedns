package config

import (
	"k8s.io/client-go/tools/clientcmd"
)

type HomeDirectory struct {
}

var _ Rule = (*HomeDirectory)(nil)

func NewHomeDirectory() Rule {
	return HomeDirectory{}
}

func (h HomeDirectory) GetConfig(opts *ConfigOptions) (*Config, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if err := rules.Migrate(); err != nil {
		return nil, err
	}

	cfg, err := TryKubeconfigFile(clientcmd.RecommendedHomeFile, opts)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
