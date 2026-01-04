package restconfig

import (
	"github.com/mandelsoft/flagutils"
	"github.com/spf13/pflag"
	"k8s.io/client-go/rest"
)

type ContextOption struct {
	name string

	context string
}

var (
	_ flagutils.Options = (*KubeConfigOption)(nil)
	_ Rule              = (*KubeConfigOption)(nil)
	_ Personalizable    = (*KubeConfigOption)(nil)
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

func (r *ContextOption) GetConfig(opts *RuleOptions) (*rest.Config, error) {
	if r.context != "" {
		opts.CurrentContext = r.context
	}
	return nil, nil
}
