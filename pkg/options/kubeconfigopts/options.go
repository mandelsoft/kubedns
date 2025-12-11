package kubeconfigopts

import (
	"context"
	"fmt"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/goutils/general"
	"github.com/mandelsoft/kubedns/pkg/kubeconfig"
	"github.com/spf13/pflag"
	"k8s.io/client-go/rest"
)

type Options struct {
	name string
	desc string

	KubeConfig string
	Context    string
	// Config override sany option setting
	Config *rest.Config
}

func From(opts flagutils.OptionSetProvider) *Options {
	return flagutils.GetFrom[*Options](opts)
}

var (
	_ flagutils.Options     = (*Options)(nil)
	_ flagutils.Validatable = (*Options)(nil)
)

func New(desc string, name ...string) *Options {
	return &Options{name: general.Optional(name...), desc: desc}
}

func (o *Options) Validate(ctx context.Context, opts flagutils.OptionSet, v flagutils.ValidationSet) error {
	if o.Config == nil {
		var err error
		o.Config, err = kubeconfig.GetConfig(o.KubeConfig, o.Context)
		if err != nil {
			if o.name != "" {
				return fmt.Errorf("%s: %w", o.name, err)
			}
			return err
		}
	}
	return nil
}

func (o *Options) option(name string) string {
	if o.name == "" {
		return name
	}
	return o.name + "-" + name
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&o.Context, o.option("context"), "", "", "context used together with "+o.option("kubeconfig"))
	fs.StringVarP(&o.KubeConfig, o.option("kubeconfig"), "", "", o.desc)
}

func (o *Options) GetRestConfig() *rest.Config {
	return o.Config
}
