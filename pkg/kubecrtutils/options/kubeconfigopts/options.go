package kubeconfigopts

import (
	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cluster/config"
	"github.com/spf13/pflag"
	"k8s.io/client-go/rest"
)

type Options struct {
	rules  config.Rules
	config *rest.Config
}

func From(opts flagutils.OptionSetProvider) *Options {
	return flagutils.GetFrom[*Options](opts)
}

var (
	_ flagutils.Options = (*Options)(nil)
)

func New(rules ...config.Rule) *Options {
	return &Options{rules: config.NewRules(rules...)}
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	o.rules.AddFlags(fs)
}

func (o *Options) GetConfig(*config.ConfigOptions) (*rest.Config, *config.ConfigOptions, error) {
	var opts config.ConfigOptions
	cfg, err := o.rules.GetConfig(&opts)
	if err != nil {
		return nil, nil, err
	}
	return cfg, &opts, nil
}
