package types

import (
	"context"

	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/enqueue"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/internal"
	"github.com/mandelsoft/logging"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/managedfields"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type ControllerDefinition interface {
}

type ControllerManager interface {
	GetName() string
	GetManager() ctrl.Manager
	GetMainCLuster() Cluster
	GetCluster(name string) Cluster
	GetClusters() Clusters
	GetIndex(name string) Index
	GetIndices() Indices

	GetLogger() logging.Logger
}

type Controller interface {
	GetName() string
	GetLogger() logging.Logger
	GetClusters() Clusters
	GetCluster() Cluster
	GetResource() client.Object
	GetControllerManager() ControllerManager
	GetRecoder() record.EventRecorder
	GetReconciler() reconcile.Reconciler
	GetIndex(name string) Index
}

type Controllers interface {
	internal.Group[Controller]
}

type Cluster interface {
	client.Client
	cluster.Cluster
	enqueue.Mux

	GetName() string
	GetEffective() Cluster
	GetCluster() cluster.Cluster
	GetIndex(name string) Index

	GetTypeConverter() managedfields.TypeConverter
	CreateIndex(ctx context.Context, name string, proto client.Object, indexer client.IndexerFunc, wrap ...func(cluster Cluster, name string) (Index, error)) (Index, error)
	ApplyTrigger(builder *ctrl.Builder, proto client.Object) error

	IsSameAs(Cluster) bool
}

type Clusters interface {
	internal.Group[Cluster]
}

type Index interface {
	GetName() string
	GetCluster() Cluster
	GetList(ctx context.Context, namespace, key string) (client.ObjectList, error)

	ForEachItem(ctx context.Context, namespace, key string, action func(object runtime.Object) error) error
	Trigger(ctx context.Context, namespace, key string) error
}

type Indices interface {
	internal.Group[Index]
}
