package up

import (
	"context"

	"github.com/mandelsoft/goutils/general"
	"github.com/mandelsoft/kubecrtutils"
	"github.com/mandelsoft/kubecrtutils/controller"
	"github.com/mandelsoft/kubecrtutils/controller/constraints"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconcile"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconciler"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconciler/factories"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/replicate"
	"github.com/mandelsoft/kubedns/internal/controller/replicate/common"
)

type ResponsibilityFactory[P kubecrtutils.ObjectPointer[T], T any] = func(c controller.TypedController[P, T]) (ResponsibilityHandler[P, T], error)

type ResponsibilityHandler[P kubecrtutils.ObjectPointer[T], T any] interface {
	Delete(r *ReconcileRequest[P, T])
	IsResponsible(*ReconcileRequest[P, T]) (bool, reconcile.Problem)
	SetResponsibility(r *ReconcileRequest[P, T], obj P)
}

func Controller[P kubecrtutils.ObjectPointer[T], T any](name, group string, mp common.MappingProvider, resp ...ResponsibilityFactory[P, T]) controller.CompositionInterface[P, T] {
	r := general.Optional(resp...)
	return controller.Define[P, T](name+".up", replicate.SOURCE,
		factories.NewByFactory[*common.Options, Settings[P, T], P, T](&Factory[P, T]{resp: r, mapprov: mp})).
		UseCluster(replicate.TARGET).
		WithFinalizer(name).
		InGroup(replicate.GROUP, group).
		WithActivationConstraint(constraints.Complete(group))
}

type Factory[P kubecrtutils.ObjectPointer[T], T any] struct {
	mapprov common.MappingProvider
	resp    ResponsibilityFactory[P, T]
	common.Factory
}

var _ factories.Factory[*common.Options, Settings[*corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry], *corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry] = (*Factory[*corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry])(nil)

func (f *Factory[P, T]) CreateSettings(ctx context.Context, o *common.Options, c controller.TypedController[P, T]) (Settings[P, T], error) {
	tgt := c.GetClusters().Get(replicate.TARGET).AsCluster()
	l := c.GetLogger()
	l.Info("creating entry down replicator...")
	l.Info("using source {{ctype}} {{cluster}}[{{info}}]", "ctype", c.GetCluster().GetTypeInfo(), "cluster", c.GetCluster().GetName(), "info", c.GetCluster().GetInfo())
	l.Info("using target {{ctype}} {{cluster}}[{{info}}]", "ctype", tgt.GetTypeInfo(), "cluster", tgt.GetName(), "info", tgt.GetInfo())

	var resp ResponsibilityHandler[P, T]
	var err error
	if f.resp != nil {
		resp, err = f.resp(c)
		if err != nil {
			return Settings[P, T]{}, err
		}
	}
	return Settings[P, T]{
		Target:  tgt,
		Mapping: f.mapprov.GetMapping(o),
		Resp:    resp,
	}, nil
}

func (f *Factory[P, T]) CreateRequest(def *reconciler.BaseRequest[P], r *factories.Reconciler[*common.Options, Settings[P, T], P, T]) reconciler.ReconcileRequest[P] {
	req := &ReconcileRequest[P, T]{
		DefaultReconcileRequest: reconciler.DefaultReconcileRequest[P, *factories.Reconciler[*common.Options, Settings[P, T], P, T]]{*def, r},
	}
	req.MappingContext = common.WithCluster(req, r.Settings.Target)
	return req

}
