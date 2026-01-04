package cluster

import (
	"context"
	"fmt"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cluster/restconfig"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/internal"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/options/kubeconfigopts"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
)

const DEFAULT = "default"

func From(opts flagutils.OptionSetProvider) Definitions {
	return flagutils.GetFrom[Definitions](opts)
}

type DefinitionProvider interface {
	GetDefinition() Definition
}

type Definition interface {
	DefinitionProvider
	flagutils.Options
	flagutils.OptionSetProvider

	GetConfig(*restconfig.RuleOptions) (*rest.Config, error)

	GetName() string
	GetFallback() string
	GetDescription() string
	GetScheme() *runtime.Scheme

	WithFallback(fallback string) Definition
	WithScheme(scheme *runtime.Scheme) Definition
}

type definition struct {
	name     string
	fallback string
	rules    restconfig.Rules
	desc     string
	scheme   *runtime.Scheme
}

var _ Definition = (*definition)(nil)

func NewDefinition(name string, desc string, rule ...restconfig.Rule) Definition {
	if len(rule) == 0 {
		rule = []restconfig.Rule{restconfig.DedicatedConfigRules(name, desc)}
	}
	return &definition{name: name, desc: desc, fallback: DEFAULT, rules: restconfig.NewRules(rule...)}
}

func (d *definition) WithFallback(fallback string) Definition {
	d.fallback = fallback
	return d
}

func (d *definition) WithScheme(scheme *runtime.Scheme) Definition {
	d.scheme = scheme
	return d
}

func (d *definition) GetDefinition() Definition {
	return d
}

func (d *definition) GetName() string {
	return d.name
}

func (d *definition) GetFallback() string {
	return d.fallback
}

func (d *definition) GetDescription() string {
	return d.desc
}

func (d *definition) GetScheme() *runtime.Scheme {
	return d.scheme
}

func (d *definition) GetConfig(*restconfig.RuleOptions) (*rest.Config, error) {
	return d.rules.GetConfig(nil)
}

func (d *definition) AddFlags(fs *pflag.FlagSet) {
	d.rules.AddFlags(fs)
}

func (d *definition) AsOptionSet() flagutils.OptionSet {
	return d.rules.AsOptionSet()
}

////////////////////////////////////////////////////////////////////////////////

type Definitions interface {
	internal.Definitions[Definition, Definitions]
	flagutils.Validatable

	WithScheme(scheme *runtime.Scheme) Definitions
	GetError() error

	GetClusters() Clusters
}

type definitions struct {
	internal.DefinitionsImpl[Definition, Definitions]
	scheme   *runtime.Scheme
	main     *kubeconfigopts.Options
	clusters Clusters
}

var _ Definitions = (*definitions)(nil)

func NewDefinitions() Definitions {
	d := &definitions{
		main: kubeconfigopts.New(),
	}
	d.DefinitionsImpl = internal.NewDefinitions[Definition, Definitions]("cluster", d)
	return d
}

func (d *definitions) WithScheme(scheme *runtime.Scheme) Definitions {
	d.scheme = scheme
	return d
}

func (d *definitions) AddFlags(fs *pflag.FlagSet) {
	d.main.AddFlags(fs)
	d.DefinitionsImpl.AddFlags(fs)
}

func (d *definitions) Validate(ctx context.Context, opts flagutils.OptionSet, v flagutils.ValidationSet) error {
	if d.clusters == nil && d.GetError() == nil {
		err := v.ValidateSet(ctx, opts, &d.DefinitionsImpl)
		if err != nil {
			return err
		}
		d.clusters = NewClusters()

		missing := true
		found := true
		for missing && found {
			missing = false
			found = false
			for n, def := range d.Elements {
				if d.clusters.Get(n) == nil {
					cfg, err := def.GetConfig(nil)
					if err != nil {
						return fmt.Errorf("cluster %s: %w", n, err)
					}
					if cfg != nil {
						cluster, err := d.newCluster(def.GetName(), def.GetScheme(), cfg)
						if err != nil {
							return err
						}
						found = true
						d.clusters.Add(cluster)
						continue
					}

					fb := def.GetFallback()
					if fb == "" {
						missing = true
						continue
					}
					eff := d.clusters.Get(fb)
					if eff != nil {
						d.clusters.Add(NewAlias(n, eff))
						found = true
					} else {
						if fb == DEFAULT {
							err := v.Validate(ctx, opts, d.main)
							if err != nil {
								return err
							}
							cfg, err := d.main.GetConfig(nil)
							if err != nil {
								return err
							}
							if cfg == nil {
								return fmt.Errorf("no default kubeconfig found, use option --kubeconfig")
							}
							cluster, err := d.newCluster(DEFAULT, d.scheme, cfg)
							if err != nil {
								return err
							}
							d.clusters.Add(cluster)
							found = true
						}
						missing = true
					}
				}
			}
		}
		if missing {
			for n, _ := range d.Elements {
				if d.clusters.Get(n) == nil {
					return fmt.Errorf("kubeconfig for cluster %q required", n)
				}
			}
		}
	}
	return d.GetError()
}

func (d *definitions) newCluster(name string, scheme *runtime.Scheme, cfg *rest.Config) (Cluster, error) {
	return NewCluster(name, cfg, func(opts *cluster.Options) {
		if scheme != nil {
			opts.Scheme = scheme
		} else {
			opts.Scheme = d.scheme
		}
	})
}

func (d *definitions) GetClusters() Clusters {
	return d.clusters
}

func ValidatedClusters(ctx context.Context, opts flagutils.OptionSet, v flagutils.ValidationSet) (Clusters, error) {
	defs, err := flagutils.ValidatedOptions[Definitions](ctx, opts, v)
	if err != nil {
		return nil, err
	}
	return defs.GetClusters(), nil
}
