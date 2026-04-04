package hostedzone

import (
	"context"

	"github.com/mandelsoft/kubecrtutils/controller"
	"github.com/mandelsoft/kubecrtutils/controller/constraints"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconciler"
	"github.com/mandelsoft/kubecrtutils/controller/support"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/server"
	"github.com/mandelsoft/kubedns/internal/controller/server/servercomp"
	"github.com/mandelsoft/kubedns/internal/controller/server/zonemodel"
)

func Controller() controller.Definition {
	return controller.Define[*corednsv1alpha1.HostedZone, corednsv1alpha1.HostedZone](server.ControllerHostedzone, server.CLUSTER,
		support.NewByFactory[support.None, *Settings, *corednsv1alpha1.HostedZone, corednsv1alpha1.HostedZone](&Factory{}),
	).
		InGroup(server.GROUP).UseComponent(server.Component).
		WithActivationConstraint(constraints.Complete(server.GROUP))
}

type Settings struct {
	model *zonemodel.Model
}

type Factory struct {
	support.DefaultFactory[support.None, *Settings, *corednsv1alpha1.HostedZone, corednsv1alpha1.HostedZone]
}

func (f *Factory) CreateSettings(ctx context.Context, o support.None, controller controller.TypedController[*corednsv1alpha1.HostedZone, corednsv1alpha1.HostedZone]) (*Settings, error) {
	return &Settings{
		model: controller.GetComponents().Get(server.Component).(*servercomp.Component).GetModel(),
	}, nil
}

func (f *Factory) CreateRequest(r *reconciler.BaseRequest[*corednsv1alpha1.HostedZone], r2 *support.Reconciler[support.None, *Settings, *corednsv1alpha1.HostedZone, corednsv1alpha1.HostedZone]) reconciler.ReconcileRequest[*corednsv1alpha1.HostedZone] {
	return &Request{r, r2.Settings.model}
}
