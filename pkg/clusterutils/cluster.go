package clusterutils

import (
	"github.com/mandelsoft/kubedns/pkg/merge"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/managedfields"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
)

type SchemeProvider interface {
	GetScheme() *runtime.Scheme
}

type Cluster interface {
	cluster.Cluster
	client.Client

	GetName() string
	GetEffective() Cluster
	GetCluster() cluster.Cluster
	GetTypeConverter() managedfields.TypeConverter
}

type _cluster struct {
	cluster.Cluster
	client.Client
	name      string
	converter managedfields.TypeConverter
}

var _ SchemeProvider = (*_cluster)(nil)

type _alias struct {
	Cluster
	name string
}

func NewAlias(name string, c Cluster) Cluster {
	return &_alias{
		Cluster: c,
		name:    name,
	}
}

func (c *_alias) GetName() string {
	return c.name
}

func NewClusterForCluster(name string, c cluster.Cluster) Cluster {
	conv, err := merge.NewConverterV3(c.GetConfig())
	if err != nil {
		return nil
	}
	return &_cluster{
		Cluster:   c,
		Client:    c.GetClient(),
		name:      name,
		converter: conv,
	}
}

func NewCluster(name string, config *rest.Config, opts ...cluster.Option) (Cluster, error) {
	c, err := cluster.New(config, opts...)
	if err != nil {
		return nil, err
	}
	conv, err := merge.NewConverterV3(config)
	if err != nil {
		return nil, err
	}
	return &_cluster{
		Cluster:   c,
		Client:    c.GetClient(),
		name:      name,
		converter: conv,
	}, nil
}

func (c *_cluster) GetName() string {
	return c.name
}

func (c *_cluster) GetEffective() Cluster {
	return c
}

func (c *_cluster) GetCluster() cluster.Cluster {
	return c.Cluster
}

func (c *_cluster) GetTypeConverter() managedfields.TypeConverter {
	return c.converter
}
