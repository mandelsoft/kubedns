package controller

import (
	"context"
	"fmt"

	"github.com/mandelsoft/kubedns/pkg/kubecrtutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cacheindex"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cluster"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/types"
	"github.com/mandelsoft/logging"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	builder2 "sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type _controller[T any, P kubecrtutils.ObjectPointer[T]] struct {
	controllerManager types.ControllerManager
	definition        TypedDefinition[T, P]
	logger            logging.Logger
	clusters          types.Clusters
	cluster           types.Cluster
	gk                schema.GroupKind
	recorder          record.EventRecorder
	indices           map[string]cacheindex.Index[T]
	reconciler        reconcile.Reconciler
}

func (c *_controller[T, P]) GetName() string {
	return c.definition.GetName()
}

func (c *_controller[T, P]) GetFieldManager() string {
	return c.controllerManager.GetName() + "/" + c.definition.GetName()
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

func (c *_controller[T, P]) GetTypedIndex(name string) cacheindex.Index[T] {
	i := c.indices[name]
	return i
}

func (c *_controller[T, P]) GetIndex(name string) cluster.Index {
	return c.GetTypedIndex(name)
}

func (c *_controller[T, P]) GetReconciler() reconcile.Reconciler {
	return c.reconciler
}

func (c *_controller[T, P]) Complete(ctx context.Context) error {
	d := c.definition
	mgr := c.GetControllerManager()
	logger := c.GetLogger()
	cl := c.GetCluster()
	builder := ctrl.NewControllerManagedBy(mgr.GetManager()).Named(d.GetName())

	if cl.IsSameAs(mgr.GetMainCluster()) {
		logger.Info("configure reconciling {{kind}} at main cluster {{cluster}}[{{effcluster}}]", "kind", c.gk, "cluster", d.GetCluster(), "effcluster", cl.GetEffective().GetName())
		builder.For(d.GetResource(), builder2.WithPredicates(d.GetWatchPredicates()...))
	} else {
		logger.Info("configure reconciling {{kind}} at cluster {{cluster}}[[[effcluster}}]", "kind", c.gk, "cluster", d.GetCluster(), "effcluster", cl.GetEffective().GetName())
		builder.WatchesRawSource(
			source.Kind(
				cl.GetCache(),
				d.GetResource(),
				&handler.EnqueueRequestForObject{},
				d.GetWatchPredicates()...,
			),
		)
	}

	trigger, err := cl.TriggerSource(d.GetResource())
	if err != nil {
		return fmt.Errorf("explicit trigger [%s]: %w", c.gk, err)
	}
	logger.Info("configure explicit trigger for main resource {{kind}} at cluster {{cluster}}[{{effcluster}}]", "kind", c.gk, "cluster", d.GetCluster(), "effcluster", c.GetCluster().GetEffective().GetName())
	builder.WatchesRawSource(trigger)

	logger.Info("configure reconciler")
	r, err := d.GetReconciler().CreateReconciler(ctx, c, builder)
	if err != nil {
		return err
	}
	c.reconciler = r

	for _, t := range d.GetTriggers() {
		err := c.addResourceTrigger(builder, t)
		if err != nil {
			return fmt.Errorf("resource-based trigger: %w", err)
		}
	}

	err = builder.Complete(r)
	if err != nil {
		return err
	}
	return nil
}

func (c *_controller[T, P]) addResourceTrigger(builder *builder2.Builder, t ResourceTriggerDefinition) error {
	d := c.definition
	mapper, cl, gk, err := t.GetMapper(c)
	if err != nil {
		return fmt.Errorf("cannot create trigger: %w", err)
	}
	if cl.IsSameAs(c.controllerManager.GetMainCluster()) {
		c.logger.Info("configure resource-based trigger {{resource}}[{{trigger}}] on main cluster {{cluster}}[{{effcluster}}]", "trigger", t.GetDescription(), "resource", *gk, "cluster", d.GetCluster(), "effcluster", cl.GetEffective().GetName())
		builder.Watches(t.GetResource(),
			handler.EnqueueRequestsFromMapFunc(mapper),
		)
	} else {
		c.logger.Info("configure resource-based trigger for {{resource}}[{{trigger}}] on cluster {{cluster}[{{effcluster}}]}", "trigger", t.GetDescription(), "resource", *gk, "cluster", d.GetCluster(), "effcluster", cl.GetEffective().GetName())
		builder.WatchesRawSource(
			source.Kind(cl.GetCache(), t.GetResource(),
				handler.EnqueueRequestsFromMapFunc(mapper),
			),
		)
	}
	return nil
}
