package cluster

import (
	"context"
	"fmt"
	"sync"

	"github.com/mandelsoft/goutils/general"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/enqueue"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/merge"
	"k8s.io/apimachinery/pkg/util/managedfields"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
)

type _cluster struct {
	lock sync.Mutex
	cluster.Cluster
	client.Client
	enqueue.Mux
	name      string
	converter managedfields.TypeConverter
	start     sync.Once
	indices   map[string]Index
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

func NewClusterForCRTCluster(name string, c cluster.Cluster) Cluster {
	conv, err := merge.NewConverterV3(c.GetConfig())
	if err != nil {
		return nil
	}
	return &_cluster{
		Cluster:   c,
		Client:    c.GetClient(),
		name:      name,
		converter: conv,
		Mux:       enqueue.NewMux(c.GetScheme()),
		indices:   map[string]Index{},
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
		indices:   map[string]Index{},
		Mux:       enqueue.NewMux(c.GetScheme()),
	}, nil
}

func (c *_cluster) GetName() string {
	return c.name
}

func (c *_cluster) IsSameAs(o Cluster) bool {
	return c.GetClient() == o.GetClient()
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

// shitty manager API always creates an own cluster
// based on a rest config,
// which is implicitly added as runnable.
// It is not possible to pass a preconfigured cluster object.
// The workaround to fake NewClient and NewCache via options
// (see in the manager creation in package manageroptions)
// would result in calling Start on the Cache twice.
// It is not possible to ignore the call to the cache,
// because it is required to pass the synchronization barrier.
// Therefore, we have to assure that the cache start starts
// the complete cluster, but only once.
// In addition, we do not add the maon cluster object to the
// manager. This is implicitly done by providing the cache of
// this cluster.
func (c *_cluster) Start(ctx context.Context) error {
	var err error
	c.start.Do(func() {
		err = c.Cluster.Start(ctx)
	})
	return err
}

func (c *_cluster) GetCache() cache.Cache {
	return &cacheWrapper{Cache: c.Cluster.GetCache(), cluster: c}
}

type cacheWrapper struct {
	cache.Cache
	cluster *_cluster
}

// Start of the cache (provided to configure the main cluster in the
// manager, now start the complete cluster it is taken from.
// This cluster is then NOT added as runnable to the manager,
// but started via the provided cache object, which is used
// to setup the implicit cluster always created by the manager.
func (w *cacheWrapper) Start(ctx context.Context) error {
	return w.cluster.Start(ctx)
}

func (c *_cluster) GetIndex(name string) Index {
	return c.indices[name]
}

func (c *_cluster) CreateIndex(ctx context.Context, name string, proto client.Object, indexer client.IndexerFunc, wrap ...func(cluster Cluster, name string) (Index, error)) (Index, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.indices[name] != nil {
		return nil, fmt.Errorf("index %q already defined", name)
	}
	err := c.GetFieldIndexer().IndexField(ctx, proto, name, indexer)
	if err != nil {
		return nil, err
	}

	var idx Index

	w := general.Optional(wrap...)
	if w != nil {
		idx, err = w(c, name)
		if err != nil {
			return nil, err
		}
	} else {
		idx, err = NewDefaultIndex(name, c, proto)
		if err != nil {
			return nil, err
		}
	}

	c.indices[name] = idx
	return idx, nil
}

func (c *_cluster) ApplyTrigger(builder *ctrl.Builder, proto client.Object) error {
	trigger, err := c.TriggerSource(proto)

	if err != nil {
		return err
	}
	builder.WatchesRawSource(trigger)
	return nil
}
