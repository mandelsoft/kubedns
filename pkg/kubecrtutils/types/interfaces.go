package types

import (
	"context"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/enqueue"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/internal"
	"github.com/mandelsoft/logging"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/managedfields"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type ControllerManager interface {
	GetName() string
	GetManager() ctrl.Manager
	GetMainCluster() Cluster
	GetCluster(name string) Cluster
	GetClusters() Clusters
	GetIndex(name string) Index
	GetIndices() Indices

	GetLogger() logging.Logger
	GetControllerDefinition(name string) ControllerDefinition
}

type ControllerDefinition interface {
	flagutils.Options
	GetName() string
	GetCluster() string
	GetClusters() sets.Set[string]
	GetResource() client.Object
	GetWatchPredicates() []predicate.Predicate

	GetError() error
	GetOptions() flagutils.Options

	// CreateController handles the global definitions and provides
	// a Controller
	CreateController(ctx context.Context, mgr ControllerManager) (Controller, error)
}

type Controller interface {
	GetName() string
	GetFieldManager() string
	GetLogger() logging.Logger
	GetClusters() Clusters
	GetCluster() Cluster
	GetResource() client.Object
	GetControllerManager() ControllerManager
	GetRecoder() record.EventRecorder
	GetReconciler() reconcile.Reconciler
	GetIndex(name string) Index

	Complete(ctx context.Context) error
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
