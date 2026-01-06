package config

import (
	"github.com/mandelsoft/flagutils"
	"github.com/spf13/pflag"
)

type ContextOption struct {
	name string

	context string
}

var (
	_ flagutils.Options = (*ContextOption)(nil)
	_ Rule              = (*ContextOption)(nil)
	_ Personalizable    = (*ContextOption)(nil)
)

func NewContextOption(name string) *ContextOption {
	if name == "" {
		name = "kubeconfig"
	}
	return &ContextOption{
		name: name,
	}
}

func (r *ContextOption) PersonalizedWith(o *Personalization) Rule {
	name := o.Name
	if len(name) == 0 {
		name = r.name
	} else {
		name = name + "-" + r.name
	}
	return &ContextOption{
		name: name,
	}
}

func (r *ContextOption) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&r.context, r.name+"-context", "", "", "context used together with "+r.name)
}

func (r *ContextOption) GetConfig(opts *ConfigOptions) (*Config, error) {
	if r.context != "" {
		opts.CurrentContext = r.context
	}
	return nil, nil
}
