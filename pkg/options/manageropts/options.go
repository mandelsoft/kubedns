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

	defaultElectionId string

	// Configurations describes a sequence of ConfigurationProvider.
	// They are used to finalize the manager options before
	// the manager is created.
	Configurations []ConfigurationProvider
	Scheme         *runtime.Scheme
	mgr            ctrl.Manager
}

func From(opts flagutils.OptionSetProvider) *Options {
	return flagutils.GetFrom[*Options](opts)
}

var (
	_ flagutils.Options     = (*Options)(nil)
	_ flagutils.Validatable = (*Options)(nil)
	_ flagutils.OptionSet   = (*Options)(nil) // forward kubeconfig options as nested set
)

func New(kube *kubeconfigopts.Options, scheme *runtime.Scheme, electionId string, configs ...ConfigurationProvider) *Options {
	nested := flagutils.DefaultOptionSet{}
	if kube == nil {
		kube = kubeconfigopts.New("standard kubeconfig")
	}
	nested = append(nested, kube)
	nested = append(nested, tlsopts.New())
	return &Options{Nested: nested, defaultElectionId: electionId, Scheme: scheme, Configurations: configs}
}

func (o *Options) Validate(ctx context.Context, opts flagutils.OptionSet, v flagutils.ValidationSet) error {
	err := flagutils.Validate(ctx, o.Nested, v)
	if err != nil {
		return err
	}
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

	configs, err := flagutils.ValidatedFilteredOptions[ConfigurationProvider](ctx, opts, v)
	if err != nil {
		return err
	}

	cfg := ctrl.Options{
		Scheme:                 o.Scheme,
		Metrics:                metrics.GetMetricsServerOpts(),
		WebhookServer:          web.GetServer(),
		HealthProbeBindAddress: o.ProbeAddr,
		LeaderElection:         o.EnableLeaderElection,
		LeaderElectionID:       o.defaultElectionId,
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
	}

	for _, conf := range configs {
		err := conf.Configure(ctx, &cfg, opts, v)
		if err != nil {
			return err
		}
	}

	if o.ElectionId != "" {
		cfg.LeaderElectionID = o.ElectionId
	}

	for _, conf := range o.Configurations {
		err := conf.Configure(ctx, &cfg, opts, v)
		if err != nil {
			return err
		}
	}
	o.mgr, err = ctrl.NewManager(kube.GetRestConfig(), cfg)
	o.mgr.GetConfig()
	return err
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.ElectionId, "leader-election-id", o.ElectionId, "Id for leader election")
	fs.StringVar(&o.ProbeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	fs.BoolVar(&o.EnableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	o.Nested.AddFlags(fs)
}

// AsOptionSet provides access o the netsed option set.
func (o *Options) AsOptionSet() flagutils.OptionSet {
	return o.Nested
}

// Options is the iterator for nested options.
func (o *Options) Options(yield func(flagutils.Options) bool) {
	o.Nested.Options(yield)
}

////////////////////////////////////////////////////////////////////////////////

func (o *Options) GetManager() ctrl.Manager {
	return o.mgr
}
