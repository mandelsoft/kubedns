package setup

import (
	"context"

	"github.com/mandelsoft/flagutils"
	"github.com/spf13/pflag"
)

func Setup(options flagutils.OptionSet, args ...string) error {
	fs := &pflag.FlagSet{}
	options.AddFlags(fs)

	err := fs.Parse(args)
	if err != nil {
		return err
	}
	err = flagutils.Validate(context.Background(), options, nil)
	return err
}
