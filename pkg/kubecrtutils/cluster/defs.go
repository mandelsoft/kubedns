package cluster

import (
	"context"
	"fmt"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cluster/config"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/internal"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
)

func From(opts flagutils.OptionSetProvider) Definitions {
	return flagutils.GetFrom[Definitions](opts)
}

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
	main     Definition
	clusters Clusters
}

var _ Definitions = (*definitions)(nil)

func NewDefinitions() Definitions {
	d := &definitions{
		main: Define(DEFAULT, "standard cluster", config.DefaultRules()),
	}
	d.DefinitionsImpl = internal.NewDefinitions[Definition, Definitions]("cluster", d)
	return d
}

func (d *definitions) WithScheme(scheme *runtime.Scheme) Definitions {
	d.scheme = scheme
	return d
}

func (d *definitions) AddFlags(fs *pflag.FlagSet) {
	if d.Len() > 1 {
		// If we work with multiple clusters we enforce the usage og identity options
		d.main.RequireIdentity()
		for _, c := range d.Elements {
			c.RequireIdentity()
		}
	}
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
					ropts := &config.ConfigOptions{}
					cfg, err := def.GetConfig(ropts)
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
							cfg, err = d.main.GetConfig(ropts)
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

func (d *definitions) newCluster(name string, scheme *runtime.Scheme, cfg *config.Config) (Cluster, error) {
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
