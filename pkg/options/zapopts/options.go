package zapopts

import (
	"flag"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/flagutils"
	"github.com/spf13/pflag"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type Options struct {
	*zap.Options
}

func From(set flagutils.OptionSet) *Options {
	return flagutils.GetFrom[*Options](set)
}

var _ flagutils.Options = (*Options)(nil)

func New(options *zap.Options) *Options {
	return &Options{options}
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	var goflags flag.FlagSet

	o.BindFlags(&goflags)
	fs.AddGoFlagSet(&goflags)
}

func (o *Options) GetLogger() logr.Logger {
	return zap.New(zap.UseFlagOptions(o.Options))
}
