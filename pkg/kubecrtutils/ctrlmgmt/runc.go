package ctrlmgmt

import (
	"context"
	"fmt"

	"github.com/mandelsoft/flagutils"
	"github.com/spf13/pflag"
	ctrl "sigs.k8s.io/controller-runtime"
)

func Setup(opts flagutils.OptionSet, def Definition, args ...string) error {
	fs := &pflag.FlagSet{}

	found := From(opts)
	if found != nil {
		if found != def {
			return fmt.Errorf("options already contain a definition")
		}
	} else {
		options := flagutils.DefaultOptionSet{}
		options.Add(opts, def)
		opts = options

	}
	opts.AddFlags(fs)

	err := fs.Parse(args)
	if err != nil {
		return err
	}
	err = flagutils.Validate(context.Background(), opts, nil)
	if err != nil {
		return err
	}
	mgr, err := def.GetControllerManager(context.Background(), opts)
	mgr.GetLogger().Info("starting manager")
	return mgr.GetManager().Start(ctrl.SetupSignalHandler())
}
