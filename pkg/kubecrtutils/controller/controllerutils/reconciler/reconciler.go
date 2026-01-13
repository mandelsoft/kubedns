package reconciler

import (
	"context"
	"reflect"
	"time"

	"github.com/go-test/deep"
	"github.com/mandelsoft/goutils/general"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cluster"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/controller"
	reconcile2 "github.com/mandelsoft/kubedns/pkg/kubecrtutils/controller/controllerutils/reconcile"
	"github.com/mandelsoft/logging"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	builder2 "sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type CRTReconciler interface {
	reconcile.Reconciler
	GetEffective() any
}

type Reconciler[T client.Object] interface {
	Request(def *BaseRequest[T]) ReconcileRequest[T]
}

type ReconcileRequest[T client.Object] interface {
	context.Context
	logging.Logger
	cluster.Cluster
	record.EventRecorder

	GetKey() client.ObjectKey
	GetObject() T
	GetOrig() T

	StatusChanged() bool
	UpdateStatus() reconcile2.Problem
	GetAfter() time.Duration

	Reconcile() reconcile2.Problem
	ReconcileDeleting() reconcile2.Problem
	ReconcileDeleted() reconcile2.Problem
}

type BaseRequest[T client.Object] struct {
	shared
	context.Context
	logging.Logger
	reconcile.Request
	Object T
	Orig   T
	After  time.Duration
}

func NewBaseRequest[T client.Object](
	ctx context.Context,
	log logging.Logger,
	c cluster.Cluster,
	o T,
	after time.Duration,
) BaseRequest[T] {

	return BaseRequest[T]{
		shared: shared{
			Cluster: c,
		},
		Context: ctx,
		Logger:  log,
		Orig:    o,
		Object:  o.DeepCopyObject().(T),
		After:   after,
	}
}

func (r *BaseRequest[T]) GetObject() T {
	return r.Object
}

func (r *BaseRequest[T]) GetOrig() T {
	return r.Orig
}

func (r *BaseRequest[T]) GetKey() client.ObjectKey {
	return r.Request.NamespacedName
}

func (r *BaseRequest[T]) GetAfter() time.Duration {
	return r.After
}

func (r *BaseRequest[T]) StatusChanged() bool {
	n := reflect.ValueOf(r.Object).Elem().FieldByName("Status").Interface()
	o := reflect.ValueOf(r.Orig).Elem().FieldByName("Status").Interface()
	diff := deep.Equal(n, o)
	if len(diff) > 0 {
		r.Info("status changed", "diff", diff)
		return true
	}
	return false
}

func (r *BaseRequest[T]) UpdateStatus() reconcile2.Problem {
	err := r.Cluster.Status().Update(r, r.Object)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return reconcile2.TemporaryProblem(err)
	}
	return nil
}

type DefaultReconcileRequest[T client.Object, R any] struct {
	BaseRequest[T]
	Reconciler R
}

var _ ReconcileRequest[client.Object] = (*DefaultReconcileRequest[client.Object, any])(nil)

func (r *DefaultReconcileRequest[T, R]) ReconcileDeleted() reconcile2.Problem {
	return nil
}

func (r *DefaultReconcileRequest[T, R]) ReconcileDeleting() reconcile2.Problem {
	return nil
}

func (r *DefaultReconcileRequest[T, R]) Reconcile() reconcile2.Problem {
	return nil
}

////////////////////////////////////////////////////////////////////////////////

type ReconciliationByReconcileFunc[P kubecrtutils.ObjectPointer[T], T any] func(request ReconcileRequest[P]) reconcile2.Problem

func (f ReconciliationByReconcileFunc[P, T]) CreateReconciler(ctx context.Context, controller controller.Controller[T, P], b *builder2.Builder) (reconcile.Reconciler, error) {
	return CRTReconcilerFor[P, T](controller, &defaultReconciler[P, T]{f}, 0), nil
}

type defaultReconciler[P kubecrtutils.ObjectPointer[T], T any] struct {
	reconcile ReconciliationByReconcileFunc[P, T]
}

func (r *defaultReconciler[P, T]) Request(base *BaseRequest[P]) ReconcileRequest[P] {
	return &DefaultReconcileRequest[P, *defaultReconciler[P, T]]{
		BaseRequest: *base,
		Reconciler:  r,
	}
}

////////////////////////////////////////////////////////////////////////////////

type shared struct {
	cluster.Cluster
	record.EventRecorder
}

type crtReconciler[T client.Object] struct {
	shared     shared
	logger     logging.Logger
	reconciler Reconciler[T]
	after      time.Duration
}

var _ CRTReconciler = (*crtReconciler[client.Object])(nil)

func CRTReconcilerFor[P kubecrtutils.ObjectPointer[T], T any](c controller.Controller[T, P], r Reconciler[P], after ...time.Duration) CRTReconciler {
	return &crtReconciler[P]{
		logger: c.GetLogger(),
		shared: shared{
			Cluster:       c.GetCluster(),
			EventRecorder: c.GetRecoder(),
		},
		reconciler: r,
		after:      general.Optional(after...),
	}
}

func (d *crtReconciler[T]) GetEffective() any {
	return d.reconciler
}

func (d *crtReconciler[T]) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	var _nil T
	var prob reconcile2.Problem

	req := BaseRequest[T]{
		shared:  d.shared,
		Context: ctx,
		Logger:  d.logger.WithName(request.String()).WithValues("object", request.NamespacedName),
		Request: request,
		After:   d.after,
	}

	req.Orig = reflect.New(reflect.TypeFor[T]().Elem()).Interface().(T)
	var effreq ReconcileRequest[T]

	err := d.shared.Cluster.Get(ctx, request.NamespacedName, req.Orig)
	if err != nil {
		if errors.IsNotFound(err) {
			req.Orig = _nil
			effreq = d.reconciler.Request(&req)
			prob = effreq.ReconcileDeleted()
		} else {
			return reconcile.Result{}, err
		}
	} else {
		req.Object = req.Orig.DeepCopyObject().(T)
		effreq = d.reconciler.Request(&req)
		if effreq.GetObject().GetDeletionTimestamp().IsZero() {
			prob = effreq.Reconcile()
		} else {
			prob = effreq.ReconcileDeleting()
		}

	}
	if effreq.StatusChanged() {
		prob = reconcile2.AggregateProblem(prob, effreq.UpdateStatus())
	}
	return reconcile2.Result(effreq, prob, effreq.GetAfter())
}
