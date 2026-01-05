package manageropts

import (
	"context"
	"fmt"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cluster"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/options/metricsopts"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/options/tlsopts"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/options/webhookopts"
	"github.com/mandelsoft/logging"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
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
}

func From(opts flagutils.OptionSetProvider) *Options {
	return flagutils.GetFrom[*Options](opts)
}

var (
	_ flagutils.Options     = (*Options)(nil)
	_ flagutils.Validatable = (*Options)(nil)
)

func New(main string, electionId string, configs ...ConfigurationProvider) *Options {
	if main == "" {
		main = cluster.DEFAULT
	}
	nested := flagutils.DefaultOptionSet{}
	nested = append(nested, tlsopts.New())
	return &Options{Nested: nested, defaultElectionId: electionId, Configurations: configs, main: main}
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

func (o *Options) Validate(ctx context.Context, opts flagutils.OptionSet, v flagutils.ValidationSet) error {
	err := flagutils.Validate(ctx, o.Nested, v)
	if err != nil {
		return err
	}

	clusters, err := cluster.ValidatedClusters(ctx, opts, v)
	if err != nil {
		return err
	}

	main := clusters.Get(o.main)
	if main == nil {
		return fmt.Errorf("could not find main cluster %q", o.main)
	}

	_, err = flagutils.ValidatedOptions[*metricsopts.Options](ctx, opts, v)
	if err != nil {
		return err
	}

	_, err = flagutils.ValidatedOptions[*webhookopts.Options](ctx, opts, v)
	if err != nil {
		return err
	}

	_, err = flagutils.ValidatedFilteredOptions[ConfigurationProvider](ctx, opts, v)
	return err
}

// AsOptionSet provides access o the nested option set.
func (o *Options) AsOptionSet() flagutils.OptionSet {
	return o.Nested
}

////////////////////////////////////////////////////////////////////////////////

func (o *Options) GetMain() string {
	return o.main
}

func (o *Options) GetManager(ctx context.Context, opts flagutils.OptionSetProvider) (ctrl.Manager, error) {
	clusters := cluster.From(opts).GetClusters()
	if clusters == nil {
		return nil, fmt.Errorf("no cluster definitions found in options")
	}

	main := clusters.Get(o.main)
	if main == nil {
		return nil, fmt.Errorf("could not find main cluster %q", o.main)
	}

	metrics := metricsopts.From(opts)
	web := webhookopts.From(opts)

	configs := flagutils.Filter[ConfigurationProvider](opts)

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
		err := conf.Configure(ctx, &cfg, opts.AsOptionSet())
		if err != nil {
			return nil, err
		}
	}

	if o.ElectionId != "" {
		cfg.LeaderElectionID = o.ElectionId
	}

	for _, conf := range o.Configurations {
		err := conf.Configure(ctx, &cfg, opts.AsOptionSet())
		if err != nil {
			return nil, err
		}
	}
	m, err := ctrl.NewManager(main.GetConfig(), cfg)
	if err != nil {
		return nil, err
	}

	if err := m.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return nil, fmt.Errorf("unable to set up health check: %w", err)
	}
	if err := m.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return nil, fmt.Errorf("unable to set up ready check: %w", err)
	}

	found := sets.New[*rest.Config]()

	for _, c := range clusters.Elements {
		rcfg := c.GetEffective().GetConfig()
		if !found.Has(rcfg) {
			found.Insert(rcfg)
			cfg.Logger.Info("adding cluster {{cluster}} -> {{effective}}", "cluster", c.GetName(), "effective", c.GetEffective().GetName())
			err = m.Add(c.GetEffective())
		} else {
			cfg.Logger.Info("cluster {{cluster}} -> {{effective}} already added", "cluster", c.GetName(), "effective", c.GetEffective().GetName())
		}
		if err != nil {
			return nil, err
		}
	}
	return m, nil
}
