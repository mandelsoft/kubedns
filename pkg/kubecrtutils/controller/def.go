package controller

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cacheindex"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cluster"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/internal"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/types"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/sets"
	builder2 "sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type ReconcilerFactory[T any, P kubecrtutils.ObjectPointer[T]] interface {
	CreateReconciler(ctx context.Context, controller Controller[T, P], b *builder2.Builder) (reconcile.Reconciler, error)
}

type ReconcilerFactoryFunc[T any, P kubecrtutils.ObjectPointer[T]] func(ctx context.Context, controller Controller[T, P], b *builder2.Builder) (reconcile.Reconciler, error)

func (f ReconcilerFactoryFunc[T, P]) CreateReconciler(ctx context.Context, controller Controller[T, P], b *builder2.Builder) (reconcile.Reconciler, error) {
	return f(ctx, controller, b)
}

////////////////////////////////////////////////////////////////////////////////

type Definition = types.ControllerDefinition

type TypedDefinition[T any, P kubecrtutils.ObjectPointer[T]] interface {
	Definition

	GetReconciler() ReconcilerFactory[T, P]
	GetTriggers() []ResourceTriggerDefinition

	AddIndex(name string, indexerFunc cacheindex.IndexerFunc[P]) TypedDefinition[T, P]
	AddTrigger(trigger ...ResourceTriggerDefinition) TypedDefinition[T, P]
	UseCluster(name ...string) TypedDefinition[T, P]
}

type _definition[T any, P kubecrtutils.ObjectPointer[T]] struct {
	internal.Element
	predicates []predicate.Predicate
	cluster    string
	clusters   sets.Set[string]
	proto      client.Object
	reconciler ReconcilerFactory[T, P]
	indices    map[string]cacheindex.TypedDefinition[T, P]
	triggers   []ResourceTriggerDefinition
	err        error
}

func DefineByFunc[T any, P kubecrtutils.ObjectPointer[T]](name string, cluster string, fac ReconcilerFactoryFunc[T, P]) TypedDefinition[T, P] {
	return Define[T, P](name, cluster, fac)
}

func Define[T any, P kubecrtutils.ObjectPointer[T]](name string, cluster string, fac ReconcilerFactory[T, P]) TypedDefinition[T, P] {
	d := &_definition[T, P]{
		Element:    internal.NewElement(name),
		cluster:    cluster,
		clusters:   sets.New[string](cluster),
		proto:      kubecrtutils.Proto[T, P](),
		reconciler: fac,
		indices:    map[string]cacheindex.TypedDefinition[T, P]{},
	}
	return d
}

func (d *_definition[T, P]) WithPredicates(preds ...predicate.Predicate) *_definition[T, P] {
	d.predicates = append(d.predicates, preds...)
	return d
}

func (d *_definition[T, P]) UseCluster(name ...string) TypedDefinition[T, P] {
	d.clusters.Insert(name...)
	return d
}

func (d *_definition[T, P]) AddIndex(name string, indexerFunc cacheindex.IndexerFunc[P]) TypedDefinition[T, P] {
	if d.indices[name] != nil {
		d.err = errors.Join(d.err, fmt.Errorf("duplicate deinition of index %q", name))
	} else {
		d.indices[name] = cacheindex.NewDefinition[T, P](GlobalControllerIndexName(d.GetName(), name), d.cluster, indexerFunc)
	}
	return d
}

func (d *_definition[T, P]) AddTrigger(trigger ...ResourceTriggerDefinition) TypedDefinition[T, P] {
	for _, t := range trigger {
		d.triggers = append(d.triggers, t)
		if t.GetCluster() != "" {
			d.clusters.Insert(t.GetCluster())
		}
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

func (d *_definition[T, P]) Validate(ctx context.Context, opts flagutils.OptionSet, v flagutils.ValidationSet) error {
	if o, ok := d.reconciler.(flagutils.Validatable); ok {
		return o.Validate(ctx, opts, v)
	}
	return v.ValidateSet(ctx, opts, d.AsOptionSet())
}

func (d *_definition[T, P]) Finalize(ctx context.Context, opts flagutils.OptionSet, v flagutils.FinalizationSet) error {
	if o, ok := d.reconciler.(flagutils.Finalizable); ok {
		return o.Finalize(ctx, opts, v)
	}
	return v.FinalizeSet(ctx, opts, d.AsOptionSet())
}

func (d *_definition[T, P]) GetError() error {
	return d.err
}

func (d *_definition[T, P]) GetCluster() string {
	return d.cluster
}

func (d *_definition[T, P]) GetClusters() sets.Set[string] {
	return maps.Clone(d.clusters)
}

func (d *_definition[T, P]) GetResource() client.Object {
	return d.proto
}

func (d *_definition[T, P]) GetWatchPredicates() []predicate.Predicate {
	return slices.Clone(d.predicates)
}

func (d *_definition[T, P]) GetReconciler() ReconcilerFactory[T, P] {
	return d.reconciler
}

func (d *_definition[T, P]) GetTriggers() []ResourceTriggerDefinition {
	return slices.Clone(d.triggers)
}

func (d *_definition[T, P]) CreateController(ctx context.Context, mgr types.ControllerManager) (types.Controller, error) {
	logger := mgr.GetLogger().WithName(d.GetName()).WithValues("controller", d.GetName())
	logger.Info("configure controller {{controller}}")

	// TODO: make controller composable
	mapping := cluster.IdMapping(d.GetClusters())
	clusters, err := cluster.Map(mgr.GetClusters(), mapping)
	if err != nil {
		return nil, err
	}

	c := clusters.Get(d.cluster)
	if c == nil {
		return nil, fmt.Errorf("cluster %q not found", d.GetName())
	}

	gk, err := kubecrtutils.GKForObject(c, d.proto)
	if err != nil {
		return nil, fmt.Errorf("main resource: %w", err)
	}

	local := map[string]cacheindex.Index[T]{}
	for n, i := range d.indices {
		idx, err := i.Apply(ctx, clusters, logger)
		if err != nil {
			return nil, fmt.Errorf("index %q[%s]: %w", n, gk, err)
		}
		mgr.GetIndices().Add(idx)
		local[n] = idx.(cacheindex.Index[T])
	}

	controller := &_controller[T, P]{
		controllerManager: mgr,
		logger:            logger,
		clusters:          clusters, // TODO; name mapping
		cluster:           c,
		gk:                gk,
		definition:        d,
		recorder:          c.GetEventRecorderFor(mgr.GetName() + "/" + c.GetName()),
		indices:           local,
	}
	return controller, nil
}

func (d *_definition[T, P]) GetOptions() flagutils.Options {
	if o, ok := d.reconciler.(flagutils.Options); ok {
		return o
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

func GlobalControllerIndexName(cname, iname string) string {
	return cname + ":" + iname
}
