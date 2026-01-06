package config

import (
	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/goutils/general"
	"github.com/spf13/pflag"
)

type IdOption struct {
	name    string
	enabled bool

	id string
}

var (
	_ flagutils.Options = (*IdOption)(nil)
	_ Rule              = (*IdOption)(nil)
	_ Personalizable    = (*IdOption)(nil)
	_ IdentityProvider  = (*IdOption)(nil)
)

func NewIdOption(name string, id ...bool) *IdOption {
	if name == "" {
		name = "kubeconfig"
	}
	return &IdOption{
		name:    name,
		enabled: general.Optional(id...),
	}
}

func (r *IdOption) RequireIdentity() {
	r.enabled = true
}

func (r *IdOption) PersonalizedWith(o *Personalization) Rule {
	name := o.Name
	if len(name) == 0 {
		name = r.name
	} else {
		name = name + "-" + r.name
	}
	return &IdOption{
		name: name,
	}
}

func (r *IdOption) AddFlags(fs *pflag.FlagSet) {
	if r.enabled {
		fs.StringVarP(&r.id, r.name+"-identity", "", "", "context used together with "+r.name)
	}
}

func (r *IdOption) GetConfig(opts *ConfigOptions) (*Config, error) {
	if r.id != "" && r.enabled {
		opts.Idenitity = r.id
	}
	return nil, nil
}
