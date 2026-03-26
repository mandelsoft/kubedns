package common

import (
	"github.com/mandelsoft/flagutils"
	"github.com/spf13/pflag"
)

type Options struct {
	Class string
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
	fs.StringVarP(&o.Class, "class", "", "", "name of the controller class to handle")
}

func Assure(opts flagutils.OptionSet) error {
	return flagutils.Assure(opts, NewOptions)
}
