package controller

import (
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cluster"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/index"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/types"
	"github.com/mandelsoft/logging"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type _controller[T any, P kubecrtutils.ObjectPointer[T]] struct {
	controllerManager types.ControllerManager
	definition        TypedDefinition[T, P]
	logger            logging.Logger
	clusters          types.Clusters
	cluster           types.Cluster
	recorder          record.EventRecorder
	indices           map[string]index.Index[T]
	reconciler        reconcile.Reconciler
}

func newController[T any, P kubecrtutils.ObjectPointer[T]](def TypedDefinition[T, P]) Controller[T, P] {
	return &_controller[T, P]{definition: def, indices: make(map[string]index.Index[T])}
}

func (c *_controller[T, P]) GetName() string {
	return c.definition.GetName()
}

func (c *_controller[T, P]) GetLogger() logging.Logger {
	return c.logger
}

func (c *_controller[T, P]) GetControllerManager() types.ControllerManager {
	return c.controllerManager
}

func (c *_controller[T, P]) GetClusters() types.Clusters {
	return c.clusters
}

func (c *_controller[T, P]) GetResource() client.Object {
	return c.definition.GetResource()
}

func (c *_controller[T, P]) GetDefinition() TypedDefinition[T, P] {
	return c.definition
}

func (c *_controller[T, P]) GetCluster() types.Cluster {
	return c.cluster
}

func (c *_controller[T, P]) GetRecoder() record.EventRecorder {
	return c.recorder
}

func (c *_controller[T, P]) GetTypedIndex(name string) index.Index[T] {
	i := c.indices[name]
	if i == nil {
	}
	return i
}

func (c *_controller[T, P]) GetIndex(name string) cluster.Index {
	return c.GetTypedIndex(name)
}

func (c *_controller[T, P]) GetReconciler() reconcile.Reconciler {
	return c.reconciler
}
