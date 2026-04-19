package generic

import (
	"context"
	"fmt"

	"github.com/mandelsoft/goutils/general"
	"github.com/mandelsoft/kubecrtutils"
	"github.com/mandelsoft/kubecrtutils/controller"
	"github.com/mandelsoft/kubecrtutils/controller/constraints"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconcile"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconciler/logic"
	"github.com/mandelsoft/kubecrtutils/examples/simple/controllers"
	"github.com/mandelsoft/kubedns/internal/controller/replicate"
	"github.com/mandelsoft/kubedns/internal/controller/replicate/common"
)

type ResponsibilityFactory[P kubecrtutils.ObjectPointer[T], T any] = func(c controller.TypedController[P, T]) (ResponsibilityHandler[P, T], error)

type ResponsibilityHandler[P kubecrtutils.ObjectPointer[T], T any] interface {
	Delete(r Request[P, T])
	IsResponsible(Request[P, T]) (bool, reconcile.Problem)
	SetResponsibility(r Request[P, T], obj P)
}

func Controller[P kubecrtutils.ObjectPointer[T], T any](name, group string, resp ...ResponsibilityFactory[P, T]) controller.CompositionInterface[P, T] {
	r := general.Optional(resp...)
	return controller.Define[P, T](name, replicate.SOURCE,
		logic.New[*common.Options, Settings[P, T], P, T](&ReconcilationLogic[P, T]{resp: r})).
		UseCluster(replicate.TARGET).
		WithFinalizer(name).
		InGroup(replicate.GROUP, group).
		AddTrigger(controller.OwnerTrigger[P]().OnCluster(controllers.TARGET)).
		WithActivationConstraint(constraints.Complete(group))
}

type ReconcilationLogic[P kubecrtutils.ObjectPointer[T], T any] struct {
	resp ResponsibilityFactory[P, T]
}

func (f *ReconcilationLogic[P, T]) CreateSettings(ctx context.Context, o *common.Options, c controller.TypedController[P, T]) (Settings[P, T], error) {
	tgt := c.GetClusters().Get(replicate.TARGET).AsCluster()
	l := c.GetLogger()
	if c.GetLogicalCluster(replicate.SOURCE) == nil {
		return Settings[P, T]{}, fmt.Errorf("%s cluster is required", replicate.SOURCE)
	}
	if c.GetLogicalCluster(replicate.TARGET) == nil {
		return Settings[P, T]{}, fmt.Errorf("%s cluster is required", replicate.TARGET)
	}
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
		Resp:    resp,
		Mapping: o.Mapping.ForResource(c.GetGroupKind()),
		Options: o,
	}, nil
}
