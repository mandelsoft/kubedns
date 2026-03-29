package down

import (
	"context"

	"github.com/mandelsoft/kubecrtutils"
	"github.com/mandelsoft/kubecrtutils/controller"
	"github.com/mandelsoft/kubecrtutils/controller/constraints"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconciler"
	"github.com/mandelsoft/kubecrtutils/controller/support"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/replicate"
	"github.com/mandelsoft/kubedns/internal/controller/replicate/common"
)

func Controller[P kubecrtutils.ObjectPointer[T], T any](name, group string, mapprov common.MappingProvider) controller.Definition {
	return controller.Define[P](name+".down", replicate.TARGET,
		support.NewByFactory[*common.Options, Settings, P](Factory[P, T]{mapprov: mapprov})).
		UseCluster(replicate.SOURCE).
		WithFinalizer(name).
		InGroup(replicate.GROUP, group).
		WithActivationConstraint(constraints.Complete(group))
}

type Factory[P kubecrtutils.ObjectPointer[T], T any] struct {
	mapprov common.MappingProvider
	common.Factory
}

var _ support.Factory[*common.Options, Settings, *corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry] = Factory[*corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry]{}

func (f Factory[P, T]) CreateSettings(ctx context.Context, o *common.Options, c controller.TypedController[P, T]) (Settings, error) {
	src := c.GetClusters().Get(replicate.SOURCE)
	l := c.GetLogger()
	l.Info("creating entry down replicator...")
	l.Info("using source {{ctype}} {{cluster}}[{info}}]", "apiserver", src.GetTypeInfo(), src.GetName(), src.GetInfo())
	l.Info("using target {{ctype}} {{cluster}}[{info}}]", "apiserver", c.GetCluster().GetTypeInfo(), c.GetCluster().GetName(), c.GetCluster().GetInfo())
	return Settings{
		Source:  src,
		Mapping: f.mapprov.GetMapping(o),
	}, nil
}

func (f Factory[P, T]) CreateRequest(def *reconciler.BaseRequest[P], r *support.Reconciler[*common.Options, Settings, P, T]) reconciler.ReconcileRequest[P] {
	req := &ReconcileRequest[P, T]{
		DefaultReconcileRequest: reconciler.DefaultReconcileRequest[P, *support.Reconciler[*common.Options, Settings, P, T]]{*def, r},
	}
	req.MappingContext = req
	return req
}
