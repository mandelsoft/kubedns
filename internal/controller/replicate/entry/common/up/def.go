package up

import (
	"context"

	"github.com/mandelsoft/kubecrtutils"
	"github.com/mandelsoft/kubecrtutils/controller"
	"github.com/mandelsoft/kubecrtutils/controller/constraints"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconciler"
	"github.com/mandelsoft/kubecrtutils/controller/support"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/replicate"
)

func Controller[P kubecrtutils.ObjectPointer[T], T any](name, group string) controller.Definition {
	return controller.Define[P, T](name+".up", replicate.SOURCE,
		support.NewByFactory[*replicate.Options, Settings, P, T](Factory[P, T]{})).
		UseCluster(replicate.TARGET).
		WithFinalizer(name).
		InGroup(replicate.GROUP, group).
		WithActivationConstraint(constraints.Complete(group))
}

type Factory[P kubecrtutils.ObjectPointer[T], T any] struct {
	replicate.Factory
}

var _ support.Factory[*replicate.Options, Settings, *corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry] = Factory[*corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry]{}

func (f Factory[P, T]) CreateSettings(ctx context.Context, o *replicate.Options, c controller.TypedController[P, T]) Settings {
	tgt := c.GetClusters().Get(replicate.TARGET).AsCluster()
	l := c.GetLogger()
	l.Info("creating entry down replicator...")
	l.Info("using source {{ctype}} {{cluster}}[{info}}]", "apiserver", c.GetCluster().GetTypeInfo(), c.GetCluster().GetName(), c.GetCluster().GetInfo())
	l.Info("using target {{ctype}} {{cluster}}[{info}}]", "apiserver", tgt.GetTypeInfo(), tgt.GetName(), tgt.GetInfo())
	return Settings{
		Target: tgt,
	}
}

func (f Factory[P, T]) CreateRequest(def *reconciler.BaseRequest[P], r *support.Reconciler[*replicate.Options, Settings, P, T]) reconciler.ReconcileRequest[P] {
	return &ReconcileRequest[P, T]{
		DefaultReconcileRequest: reconciler.DefaultReconcileRequest[P, *support.Reconciler[*replicate.Options, Settings, P, T]]{*def, r},
	}
}
