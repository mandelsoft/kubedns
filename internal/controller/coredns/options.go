package coredns

import (
	"context"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubedns/pkg/options/kubeconfigopts"
	"github.com/spf13/pflag"
	"k8s.io/client-go/rest"
)

type Options struct {
	Runtime    string
	KubeConfig *kubeconfigopts.Options
}

var (
	_ flagutils.Options     = (*Options)(nil)
	_ flagutils.Validatable = (*Options)(nil)
)

func NewControllerOptions() *Options {
	return &Options{
		KubeConfig: kubeconfigopts.New("use separated runtime cluster", "runtime"),
	}
}

func (o *Options) Validate(ctx context.Context, opts flagutils.OptionSet, v flagutils.ValidationSet) error {
	return o.KubeConfig.Validate(ctx, opts, v)
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	o.KubeConfig.AddFlags(fs)
	fs.StringVarP(&o.Runtime, "runtime", "", "", "name of the runtime class to handle")
}

func (o *Options) GetRestConfig() *rest.Config {
	return o.KubeConfig.GetRestConfig()
}
