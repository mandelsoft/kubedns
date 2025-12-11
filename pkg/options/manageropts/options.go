package manageropts

import (
	"context"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubedns/pkg/options/kubeconfigopts"
	"github.com/mandelsoft/kubedns/pkg/options/metricsopts"
	"github.com/mandelsoft/kubedns/pkg/options/tlsopts"
	"github.com/mandelsoft/kubedns/pkg/options/webhookopts"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Options struct {
	Nested               flagutils.OptionSet
	EnableLeaderElection bool
	ProbeAddr            string
	ElectionId           string

	Scheme *runtime.Scheme
	mgr                  ctrl.Manager
}

func From(opts flagutils.OptionSetProvider) *Options {
	return flagutils.GetFrom[*Options](opts)
}

var (
	_ flagutils.Options   = (*Options)(nil)
	_ flagutils.Validatable   = (*Options)(nil)
	_ flagutils.OptionSet = (*Options)(nil) // forward kubeconfig options as nested set
)

func New(kube *kubeconfigopts.Options, scheme *runtime.Scheme, electionId string) *Options {
	nested := flagutils.DefaultOptionSet{}
	if kube != nil {
		nested = append(nested, kube)
	}
	nested = append(nested, tlsopts.New())
	return &Options{Nested: nested, ElectionId: electionId, Scheme: scheme}
}

func (o *Options) Validate(ctx context.Context, opts flagutils.OptionSet, v flagutils.ValidationSet) error {
	metrics, err := flagutils.ValidatedOptions[*metricsopts.Options](ctx, opts, v)
	if err != nil {
		return err
	}

	web, err := flagutils.ValidatedOptions[*webhookopts.Options](ctx, opts, v)
	if err != nil {
		return err
	}

	kube, err := flagutils.ValidatedOptions[*kubeconfigopts.Options](ctx, opts, v)
	if err != nil {
		return err
	}


	o.mgr, err = ctrl.NewManager(kube.GetRestConfig(), ctrl.Options{
		Scheme:                 o.Scheme,
		Metrics:                metrics.GetMetricsServerOpts(),
		WebhookServer:          web.GetServer(),
		HealthProbeBindAddress: o.ProbeAddr,
		LeaderElection:         o.EnableLeaderElection,
		LeaderElectionID:       o.ElectionId,
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	return err
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.ElectionId, "leader-election-id", o.ElectionId, "Id for leader election")
	fs.StringVar(&o.ProbeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	fs.BoolVar(&o.EnableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
}


func (o *Options) GetManager() ctrl.Manager {
	return o.mgr
}

func (o *Options) AsOptionSet() flagutils.OptionSet {
	return o.Nested
}

func (o *Options) Options(yield func(flagutils.Options) bool) {
	o.Nested.Options(yield)
}
