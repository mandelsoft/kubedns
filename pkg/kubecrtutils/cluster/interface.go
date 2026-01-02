package cluster

import (
	"context"

	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/enqueue"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/managedfields"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
)

type SchemeProvider interface {
	GetScheme() *runtime.Scheme
}

type Cluster interface {
	cluster.Cluster
	client.Client
	enqueue.Mux

	GetName() string
	GetEffective() Cluster
	GetCluster() cluster.Cluster

	GetTypeConverter() managedfields.TypeConverter
	CreateIndex(ctx context.Context, name string, proto client.Object, indexer client.IndexerFunc, wrap ...func(cluster Cluster, name string) (Index, error)) (Index, error)

	IsSameAs(Cluster) bool
}

type Index interface {
	GetName() string
	GetCluster() Cluster
	GetList(ctx context.Context, namespace, key string) (client.ObjectList, error)

	ForEachListItem(ctx context.Context, namespace, key string, action func(object runtime.Object) error) error
	Trigger(ctx context.Context, namespace, key string) error
}
