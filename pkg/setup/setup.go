package setup

import (
	"context"

	"github.com/mandelsoft/flagutils"
	"github.com/spf13/pflag"
)

func Setup(options flagutils.OptionSet, args ...string) {
	fs := &pflag.FlagSet{}
	options.AddFlags(fs)
	err := fs.Parse(args)
	if err != nil {
		ExitIfErr(err, "parsing arguments")
	}
	err = flagutils.Validate(context.Background(), options, nil)
	//ctrl.SetLogger(zapopts.From(options).GetLogger())
	if err != nil {
		ExitIfErr(err, "validation failed")
	}
}
