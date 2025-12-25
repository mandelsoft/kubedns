package clusterutils

import (
	"context"
	"fmt"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubedns/pkg/options/kubeconfigopts"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
)

type DefinitionProvider interface {
	GetDefinition() Definition
}

type Definition interface {
	DefinitionProvider

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
	desc     string
	scheme   *runtime.Scheme
}

var _ Definition = (*definition)(nil)

func NewDefinition(name string, desc string) Definition {
	return &definition{name: name, desc: desc}
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

////////////////////////////////////////////////////////////////////////////////

type Definitions interface {
	flagutils.Options
	flagutils.Validatable

	GetClusters() Clusters
}

type definitions struct {
	scheme      *runtime.Scheme
	main        *kubeconfigopts.Options
	opts        map[string]*kubeconfigopts.Options
	definitions map[string]Definition
	clusters    Clusters
}

var _ Definitions = (*definitions)(nil)

func NewDefinitions() Definitions {
	return &definitions{
		main:        kubeconfigopts.New("standard kubeconfig"),
		definitions: make(map[string]Definition),
		opts:        map[string]*kubeconfigopts.Options{},
	}
}

func (d *definitions) AddFlags(fs *pflag.FlagSet) {
	d.main.AddFlags(fs)

	for n, def := range d.definitions {
		o := d.opts[n]
		if o == nil {
			o = kubeconfigopts.New(fmt.Sprintf("kubeconfig for logical cluster %q: %s", n, def.GetDescription()), n)
			d.opts[n] = o
		}
		o.AddFlags(fs)
	}
}

func (d *definitions) Validate(ctx context.Context, opts flagutils.OptionSet, v flagutils.ValidationSet) error {
	if d.clusters == nil {
		d.clusters = NewClusters()
		for n, def := range d.definitions {
			o := d.opts[n]
			if o.KubeConfig != "" {
				err := o.Validate(ctx, opts, v)
				if err != nil {
					return err
				}
				cluster, err := d.newCluster(def.GetName(), def.GetScheme(), o.GetRestConfig())
				if err != nil {
					return err
				}
				d.clusters.Add(cluster)
			}
		}
		missing := true
		found := true
		for missing && found {
			missing = false
			found = false
			for n, def := range d.definitions {
				if d.clusters.Get(n) == nil {
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
						if fb == "default" {
							err := d.main.Validate(ctx, opts, v)
							if err != nil {
								return err
							}
							cluster, err := d.newCluster("default", nil, d.main.GetRestConfig())
							if err != nil {
								return err
							}
							d.clusters.Add(cluster)
						}
						missing = true
					}
				}
			}
			if missing {
				for n, _ := range d.definitions {
					if d.clusters.Get(n) == nil {
						return fmt.Errorf("kubeconfig for cluster %q required", n)
					}
				}
			}
		}
	}
	return nil
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
	// TODO implement me
	panic("implement me")
}

////////////////////////////////////////////////////////////////////////////////

type Clusters interface {
	Get(name string) Cluster

	Add(c Cluster)
}

type clusters struct {
	clusters map[string]Cluster
}

var _ Clusters = (*clusters)(nil)

func NewClusters() Clusters {
	return &clusters{make(map[string]Cluster)}
}

func (c *clusters) Get(name string) Cluster {
	return c.clusters[name]
}

func (c *clusters) Add(cluster Cluster) {
	c.clusters[cluster.GetName()] = cluster
}
