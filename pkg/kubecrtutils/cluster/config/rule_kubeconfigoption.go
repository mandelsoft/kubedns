package config

import (
	"strings"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/goutils/general"
	"github.com/mandelsoft/goutils/maputils"
	"github.com/spf13/pflag"
)

type KubeConfigOption struct {
	name    string
	desc    string
	special map[string]Rule

	path string

	err error
}

var (
	_ flagutils.Options = (*KubeConfigOption)(nil)
	_ Rule              = (*KubeConfigOption)(nil)
	_ Personalizable    = (*KubeConfigOption)(nil)
)

func NewKubeconfigOption(name, desc string) *KubeConfigOption {
	if name == "" {
		name = "kubeconfig"
	}
	if desc == "" {
		desc = "path to standard kubeconfig"
	}
	return &KubeConfigOption{
		name:    name,
		desc:    desc,
		special: make(map[string]Rule),
	}
}

// WithSpecialCase adds a configurable key mapped to the usage of a special rule.
// ATTENTION: This rule must not use options anymore.
func (r *KubeConfigOption) WithSpecialCase(name string, rule Rule) *KubeConfigOption {
	r.special[name] = rule
	return r
}

func (r *KubeConfigOption) PersonalizedWith(o *Personalization) Rule {
	desc := r.desc
	name := o.Name
	if o.Description != "" {
		desc = o.Description
	}
	if len(name) == 0 {
		name = r.name
	} else {
		name = name + "-" + r.name
	}
	return &KubeConfigOption{
		name:    name,
		desc:    desc,
		special: maputils.TransformValues(r.special, func(r Rule) Rule { return PersonalizeRule(r, o) }),
	}
}

func (r *KubeConfigOption) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&r.path, r.name, "", "", r.desc)
}

func (r *KubeConfigOption) GetConfig(opts *ConfigOptions) (*Config, error) {
	var cfg *Config

	if r.path != "" {
		path := r.path
		context := opts.CurrentContext
		if s := r.special[path]; s != nil {
			cfg, r.err = s.GetConfig(opts)
			return cfg, r.err
		}
		i := strings.Index(path, "@")
		if i >= 0 {
			context = path[:i]
			path = path[i+1:]
		}
		if context != "" {
			opts.CurrentContext = context
		}
		cfg, r.err = TryKubeconfigFile(path, opts)
	}
	return cfg, r.err
}

func (r *KubeConfigOption) WithInClusterMode(name ...string) *KubeConfigOption {
	return r.WithSpecialCase(general.OptionalDefaulted("in-cluster", name...), NewInClusterConfig())
}
