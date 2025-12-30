package manageropts

import (
	"context"
	"fmt"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubedns/pkg/clusterutils"
	"github.com/mandelsoft/kubedns/pkg/options/metricsopts"
	"github.com/mandelsoft/kubedns/pkg/options/tlsopts"
	"github.com/mandelsoft/kubedns/pkg/options/webhookopts"
	"github.com/mandelsoft/logging"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Options struct {
	// main is the cluster used as main cluster for the manager
	main                    string
	Nested                  flagutils.OptionSet
	EnableLeaderElection    bool
	LeaderElectionNamespace string
	ProbeAddr               string
	ElectionId              string

	defaultElectionId string

	// Configurations describes a sequence of ConfigurationProvider.
	// They are used to finalize the manager options before
	// the manager is created.
	Configurations []ConfigurationProvider
	mgmt           ControllerManagerContext
}

func From(opts flagutils.OptionSetProvider) *Options {
	return flagutils.GetFrom[*Options](opts)
}

var (
	_ flagutils.Options     = (*Options)(nil)
	_ flagutils.Validatable = (*Options)(nil)
	_ flagutils.OptionSet   = (*Options)(nil) // forward kubeconfig options as nested set
)

func New(main string, electionId string, configs ...ConfigurationProvider) *Options {
	if main == "" {
		main = clusterutils.DEFAULT
	}
	nested := flagutils.DefaultOptionSet{}
	nested = append(nested, tlsopts.New())
	return &Options{Nested: nested, defaultElectionId: electionId, Configurations: configs, main: main}
}

func (o *Options) Validate(ctx context.Context, opts flagutils.OptionSet, v flagutils.ValidationSet) error {
	if o.mgmt.Manager != nil {
		return nil
	}

	err := flagutils.Validate(ctx, o.Nested, v)
	if err != nil {
		return err
	}

	o.mgmt.Clusters, err = clusterutils.ValidatedClusters(ctx, opts, v)
	if err != nil {
		return err
	}

	main := o.mgmt.Get(o.main)
	if main == nil {
		return fmt.Errorf("could not find main cluster %q", o.main)
	}

	metrics, err := flagutils.ValidatedOptions[*metricsopts.Options](ctx, opts, v)
	if err != nil {
		return err
	}

	web, err := flagutils.ValidatedOptions[*webhookopts.Options](ctx, opts, v)
	if err != nil {
		return err
	}

	configs, err := flagutils.ValidatedFilteredOptions[ConfigurationProvider](ctx, opts, v)
	if err != nil {
		return err
	}

	cfg := ctrl.Options{
		Logger:                  logging.DefaultContext().Logger(logging.NewRealm("controller-manager")).V(4),
		Scheme:                  main.GetScheme(),
		Metrics:                 metrics.GetMetricsServerOpts(),
		WebhookServer:           web.GetServer(),
		HealthProbeBindAddress:  o.ProbeAddr,
		LeaderElection:          o.EnableLeaderElection,
		LeaderElectionNamespace: o.LeaderElectionNamespace,
		LeaderElectionID:        o.defaultElectionId,
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

		// implicit cluster creation cannot be circumvented (why), so fake
		// using shared info as far as possible.,
		NewClient: func(config *rest.Config, options client.Options) (client.Client, error) {
			return main.GetClient(), nil
		},
		NewCache: func(config *rest.Config, opts cache.Options) (cache.Cache, error) {
			return main.GetCache(), nil
		},
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
	o.mgmt.Manager, err = ctrl.NewManager(main.GetConfig(), cfg)
	if err != nil {
		return err
	}

	found := sets.New[*rest.Config]()

	for c := range o.mgmt.Clusters.Clusters {
		rcfg := c.GetEffective().GetConfig()
		if !found.Has(rcfg) {
			found.Insert(rcfg)
			cfg.Logger.Info("adding cluster {{cluster}} -> {{effective}}", "cluster", c.GetName(), "effective", c.GetEffective().GetName())
			err = o.mgmt.Manager.Add(c.GetEffective())
		} else {
			cfg.Logger.Info("cluster {{cluster}} -> {{effective}} already added", "cluster", c.GetName(), "effective", c.GetEffective().GetName())
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.ElectionId, "leader-election-id", o.ElectionId, "Id for leader election")
	fs.StringVar(&o.ProbeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	fs.StringVar(&o.LeaderElectionNamespace, "leader-elect-namespace", "", "leader election namespace")
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
	return o.mgmt.Manager
}

func (o *Options) ControllerManagerContext() *ControllerManagerContext {
	return &o.mgmt
}

////////////////////////////////////////////////////////////////////////////////

type ControllerManagerContext struct {
	ctrl.Manager
	clusterutils.Clusters
}

func (m *ControllerManagerContext) NewController() *ctrl.Builder {
	return ctrl.NewControllerManagedBy(m.Manager)
}

////////////////////////////////////////////////////////////////////////////////

func GetManagementContext(opts flagutils.OptionSetProvider) *ControllerManagerContext {
	return From(opts).ControllerManagerContext()
}
