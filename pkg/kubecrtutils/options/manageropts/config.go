package manageropts

import (
	"context"

	"github.com/mandelsoft/flagutils"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ConfigurationProvider is used to modify the manager options used to
// create a new manager.
// Such objects can be set at the Options object to preprocess the configuration
// after the option parsing. Or this interface can be implemented by other
// Options types to incorporate their settings into the configuration.
type ConfigurationProvider interface {
	Configure(ctx context.Context, config *ctrl.Options, opts flagutils.OptionSet) error
}
