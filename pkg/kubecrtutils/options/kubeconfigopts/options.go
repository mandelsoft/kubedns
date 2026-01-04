package kubeconfigopts

import (
	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cluster/restconfig"
	"github.com/spf13/pflag"
	"k8s.io/client-go/rest"
)

type Options struct {
	rules  restconfig.Rules
	config *rest.Config
}

func From(opts flagutils.OptionSetProvider) *Options {
	return flagutils.GetFrom[*Options](opts)
}

var (
	_ flagutils.Options = (*Options)(nil)
)

func New(rules ...restconfig.Rule) *Options {
	if len(rules) == 0 {
		rules = []restconfig.Rule{restconfig.DefaultRules()}
	}
	return &Options{rules: restconfig.NewRules(rules...)}
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	o.rules.AddFlags(fs)
}

func (o *Options) GetConfig(*restconfig.RuleOptions) (*rest.Config, error) {
	return o.rules.GetConfig(nil)
}
