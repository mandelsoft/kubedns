package controller

import (
	"context"
	"fmt"

	"github.com/mandelsoft/goutils/general"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/owner"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/types"
	"github.com/mandelsoft/logging"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type ResourceTriggerDefinition interface {
	OnCluster(name string) ResourceTriggerDefinition

	GetDescription() string
	GetResource() client.Object
	GetMapper(controller types.Controller) (handler.MapFunc, types.Cluster, *schema.GroupKind, error)
	GetCluster() string
}

type MapFunFactoryFunc = func(controller types.Controller, target types.Cluster, proto client.Object, log logging.Logger) (handler.MapFunc, error)
type TypedMapFuncFactoryFunc[T any] = func(controller types.Controller, target types.Cluster, proto client.Object, log logging.Logger) (handler.TypedMapFunc[T, reconcile.Request], error)

type _trigger struct {
	desc    string
	proto   client.Object
	mapper  MapFunFactoryFunc
	cluster string
}

func newTrigger[T any, P kubecrtutils.ObjectPointer[T]](mapper MapFunFactoryFunc, desc ...string) *_trigger {
	var resource T
	return &_trigger{
		desc:   general.OptionalDefaulted("resource mapping", desc...),
		proto:  any(&resource).(P),
		mapper: mapper,
	}
}

func ResourceTrigger[T any, P kubecrtutils.ObjectPointer[T]](mapFunc handler.TypedMapFunc[P, reconcile.Request], desc ...string) ResourceTriggerDefinition {
	return newTrigger[T, P](
		func(controller types.Controller, target types.Cluster, proto client.Object, log logging.Logger) (handler.MapFunc, error) {
			return ConvertMapFunc(mapFunc), nil
		},
		desc...)
}

func ResourceTriggerByFactory[T any, P kubecrtutils.ObjectPointer[T]](factoryFunc TypedMapFuncFactoryFunc[P], desc ...string) ResourceTriggerDefinition {
	return newTrigger[T, P](ConvertTriggerFunc[P](factoryFunc), desc...)
}

func OwnerTrigger[T any, P kubecrtutils.ObjectPointer[T]](fac ...owner.RemoteFactory) ResourceTriggerDefinition {
	return newTrigger[T, P](
		func(controller types.Controller, target types.Cluster, proto client.Object, log logging.Logger) (handler.MapFunc, error) {
			handler := owner.For(controller.GetCluster(), target, fac...)
			return owner.MapOwnerToLocalRequestByObject[client.Object, client.Object](handler, controller.GetCluster(), proto, log), nil
		},
		"owner trigger",
	)
}

func (t *_trigger) OnCluster(cluster string) ResourceTriggerDefinition {
	t.cluster = cluster
	return t
}

func (t *_trigger) GetResource() client.Object {
	return t.proto
}

func (t *_trigger) GetDescription() string {
	return t.desc
}

func (t *_trigger) GetMapper(controller types.Controller) (handler.MapFunc, types.Cluster, *schema.GroupKind, error) {
	c := controller.GetCluster()
	if t.cluster != "" {
		c = controller.GetClusters().Get(t.cluster)
		if c == nil {
			return nil, nil, nil, fmt.Errorf("cluster %q for trigger not defined", t.cluster)
		}
	}

	gk, err := kubecrtutils.GKForObject(c, t.proto)
	if err != nil {
		return nil, nil, nil, err
	}
	logger := controller.GetLogger().WithName("trigger").WithName(fmt.Sprintf("%s", gk))
	m, err := t.mapper(controller, c, controller.GetResource(), logger)
	if err != nil {
		return nil, nil, nil, err
	}
	return func(ctx context.Context, object client.Object) []reconcile.Request {
		r := m(ctx, object)
		if len(r) > 0 {
			logger.Info("trigger {{keys}} by resource {{key}}[{{kind}}]", "kind", gk, "keys", r, "key", client.ObjectKeyFromObject(object))
		}
		return r
	}, c, &gk, nil
}

func (t *_trigger) GetCluster() string {
	return t.cluster
}

func ConvertTriggerFunc[P client.Object](factoryFunc TypedMapFuncFactoryFunc[P]) MapFunFactoryFunc {
	return func(controller types.Controller, target types.Cluster, proto client.Object, log logging.Logger) (handler.MapFunc, error) {
		f, err := factoryFunc(controller, target, proto, log)
		if err != nil {
			return nil, err
		}
		return ConvertMapFunc(f), nil
	}
}

func ConvertMapFunc[P client.Object](mapFunc handler.TypedMapFunc[P, reconcile.Request]) handler.MapFunc {
	return func(ctx context.Context, object client.Object) []reconcile.Request {
		return mapFunc(ctx, any(object).(P))
	}
}
