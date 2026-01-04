package controller

import (
	"context"
	"fmt"

	"github.com/mandelsoft/kubedns/pkg/kubecrtutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/owner"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/types"
	"github.com/mandelsoft/logging"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type ResourceTriggerDefinition interface {
	OnCluster(name string) ResourceTriggerDefinition

	GetResource() client.Object
	GetMapper(controller types.Controller) (handler.MapFunc, types.Cluster, error)
	GetCluster() string
}

type _trigger struct {
	proto   client.Object
	mapper  func(owner types.Cluster, target types.Cluster, proto client.Object, log logging.Logger) (handler.MapFunc, error)
	cluster string
}

func ResourceTrigger[T any, P kubecrtutils.ObjectPointer[P]](mapFunc handler.TypedMapFunc[P, reconcile.Request]) ResourceTriggerDefinition {
	var resource T
	return &_trigger{
		proto: any(&resource).(P),
		mapper: func(owner, target types.Cluster, proto client.Object, log logging.Logger) (handler.MapFunc, error) {
			gk, err := kubecrtutils.GKForObject(target, any(&resource).(P))
			if err != nil {
				return nil, err
			}
			return func(ctx context.Context, object client.Object) []reconcile.Request {
				r := mapFunc(ctx, object.(P))
				if len(r) > 0 {
					log.Info("trigger {{keys}} by resource {{kind}} {{key}}", "keys", r, "kind", gk, "key", client.ObjectKeyFromObject(object))
				}
				return r
			}, nil
		},
	}
}

func OwnerTrigger[T any, P kubecrtutils.ObjectPointer[T]](fac ...owner.RemoteFactory) ResourceTriggerDefinition {
	var proto T
	return &_trigger{
		proto: any(&proto).(P),
		mapper: func(main, target types.Cluster, proto client.Object, log logging.Logger) (handler.MapFunc, error) {
			handler := owner.For(main, target, fac...)
			return owner.MapOwnerToLocalRequestByObject[client.Object, client.Object](handler, main, proto, log), nil
		},
	}
}

func (t *_trigger) OnCluster(cluster string) ResourceTriggerDefinition {
	t.cluster = cluster
	return t
}

func (t *_trigger) GetResource() client.Object {
	return t.proto
}

func (t *_trigger) GetMapper(controller types.Controller) (handler.MapFunc, types.Cluster, error) {
	c := controller.GetCluster()
	if t.cluster != "" {
		c = controller.GetClusters().Get(t.cluster)
		if c == nil {
			return nil, nil, fmt.Errorf("cluster %q for trigger not defined", t.cluster)
		}
	}
	m, err := t.mapper(controller.GetCluster(), c, controller.GetResource(), controller.GetLogger())
	if err != nil {
		return nil, nil, err
	}
	return m, c, nil
}

func (t *_trigger) GetCluster() string {
	return t.cluster
}
