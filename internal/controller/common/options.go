package common

import (
	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/flagutils/pflags"
	"github.com/spf13/pflag"
)

type Options struct {
	Class   *string
	Runtime *string
}

func From(opts flagutils.OptionSetProvider) *Options {
	return flagutils.GetFrom[*Options](opts)
}

var (
	_ flagutils.Options = (*Options)(nil)
)

func NewOptions() *Options {
	return &Options{}
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	pflags.StringRefVarP(fs, &o.Class, "class", "", nil, "name of the controller class to handle")
	pflags.StringRefVarP(fs, &o.Runtime, "runtime", "", nil, "name of the runtime to handle")
}

func Assure(opts flagutils.OptionSet) error {
	return flagutils.Assure(opts, NewOptions)
}
