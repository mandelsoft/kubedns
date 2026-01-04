package controller

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/index"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/internal"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/types"
	"github.com/spf13/pflag"
	ctrl "sigs.k8s.io/controller-runtime"
	builder2 "sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type ReconcilerFactory[T any, P kubecrtutils.ObjectPointer[T]] interface {
	CreateReconciler(ctx context.Context, controller Controller[T, P], b *builder2.Builder) (reconcile.Reconciler, error)
}

////////////////////////////////////////////////////////////////////////////////

type Definition interface {
	flagutils.Options
	GetName() string
	GetCluster() string
	GetResource() client.Object

	GetError() error

	Apply(ctx context.Context, mgr types.ControllerManager) (types.Controller, error)
}

type TypedDefinition[T any, P kubecrtutils.ObjectPointer[T]] interface {
	Definition

	GetReconciler() ReconcilerFactory[T, P]

	AddIndex(name string, indexerFunc index.IndexerFunc[P]) TypedDefinition[T, P]
	AddTrigger(trigger ...ResourceTriggerDefinition) TypedDefinition[T, P]
}

type _definition[T any, P kubecrtutils.ObjectPointer[T]] struct {
	internal.Element
	cluster    string
	proto      client.Object
	reconciler ReconcilerFactory[T, P]
	indices    map[string]index.TypedDefinition[T, P]
	triggers   []ResourceTriggerDefinition
	err        error
}

func NewDefinition[T, any, P kubecrtutils.ObjectPointer[T]](name string, cluster string, fac ReconcilerFactory[T, P]) TypedDefinition[T, P] {
	var p T
	tType := reflect.TypeOf(p)
	if tType.Kind() != reflect.Ptr {
		tType = tType.Elem()
	}
	baseType := tType.Elem()

	proto := reflect.New(baseType).Interface().(client.Object)

	return &_definition[T, P]{
		Element:    internal.NewElement(name),
		cluster:    cluster,
		proto:      proto,
		reconciler: fac,
		indices:    map[string]index.TypedDefinition[T, P]{},
	}
}

func (d *_definition[T, P]) AddIndex(name string, indexerFunc index.IndexerFunc[P]) TypedDefinition[T, P] {
	if d.indices[name] != nil {
		d.err = errors.Join(d.err, fmt.Errorf("duplicate deinition of index %q", name))
	} else {
		d.indices[name] = index.NewDefinition[T, P](d.GetName()+":"+name, d.cluster, indexerFunc)
	}
	return d
}

func (d *_definition[T, P]) AddTrigger(trigger ...ResourceTriggerDefinition) TypedDefinition[T, P] {
	for _, t := range trigger {
		d.triggers = append(d.triggers, t)
	}
	return d
}

func (d *_definition[T, P]) AddFlags(fs *pflag.FlagSet) {
	if o, ok := d.reconciler.(flagutils.Options); ok {
		o.AddFlags(fs)
	}
}

func (d *_definition[T, P]) AsOptionSet() flagutils.OptionSet {
	if o, ok := d.reconciler.(flagutils.OptionSetProvider); ok {
		return o.AsOptionSet()
	}
	return flagutils.DefaultOptionSet{}
}

func (d *_definition[T, P]) GetError() error {
	return d.err
}

func (d *_definition[T, P]) GetCluster() string {
	return d.cluster
}

func (d *_definition[T, P]) GetResource() client.Object {
	return d.proto
}

func (d *_definition[T, P]) GetReconciler() ReconcilerFactory[T, P] {
	return d.reconciler
}

func (d *_definition[T, P]) Apply(ctx context.Context, mgr types.ControllerManager) (types.Controller, error) {
	c := mgr.GetCluster(d.cluster)
	if c == nil {
		return nil, fmt.Errorf("cluster %q not found", d.GetName())
	}

	logger := mgr.GetLogger().WithName(d.GetName())
	local := map[string]index.Index[T]{}
	for n, i := range d.indices {
		idx, err := i.Apply(ctx, mgr.GetClusters())
		if err != nil {
			return nil, err
		}
		local[n] = idx.(index.Index[T])
	}
	builder := ctrl.NewControllerManagedBy(mgr.GetManager()).Named(d.GetName())

	if c.IsSameAs(mgr.GetMainCLuster()) {
		builder.For(d.proto)
	} else {
		builder.WatchesRawSource(
			source.Kind(
				c.GetCache(),
				d.proto,
				&handler.EnqueueRequestForObject{},
			),
		)
	}

	controller := &_controller[T, P]{
		controllerManager: mgr,
		logger:            logger,
		clusters:          mgr.GetClusters(), // TODO; name mapping
		cluster:           c,
		definition:        d,
		recorder:          c.GetEventRecorderFor(mgr.GetName() + "/" + c.GetName()),
		indices:           local,
	}

	trigger, err := c.TriggerSource(d.proto)
	if err != nil {
		return nil, err
	}
	builder.WatchesRawSource(trigger)

	r, err := d.reconciler.CreateReconciler(ctx, controller, builder)
	if err != nil {
		return nil, err
	}
	controller.reconciler = r

	for _, t := range d.triggers {
		err := d.addResourceTrigger(controller, builder, t)
		if err != nil {
			return nil, err
		}
	}

	err = builder.Complete(r)
	if err != nil {
		return nil, err
	}
	return controller, nil
}

func (d *_definition[T, P]) addResourceTrigger(controller *_controller[T, P], builder *builder2.Builder, t ResourceTriggerDefinition) error {
	mapper, cl, err := t.GetMapper(controller)
	if err != nil {
		return fmt.Errorf("cannot create trigger: %w", err)
	}
	if cl.IsSameAs(controller.controllerManager.GetMainCLuster()) {
		builder.Watches(t.GetResource(),
			handler.EnqueueRequestsFromMapFunc(mapper),
		)
	} else {
		builder.WatchesRawSource(
			source.Kind(cl.GetCache(), t.GetResource(),
				handler.EnqueueRequestsFromMapFunc(mapper),
			),
		)
	}
	return nil
}
