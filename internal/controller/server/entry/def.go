package entry

import (
	"github.com/mandelsoft/kubecrtutils/controller"
	"github.com/mandelsoft/kubecrtutils/controller/constraints"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconciler"
	"github.com/mandelsoft/kubecrtutils/controller/support"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/server"
	"github.com/mandelsoft/kubedns/internal/controller/server/zonemodel"
)

func Controller() controller.Definition {
	return controller.Define[*corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry](server.ControllerEntry, server.CLUSTER,
		support.NewByFactory[support.None, *Settings, *corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry](&Factory{}),
	).
		InGroup(server.GROUP).
		UseComponent(server.Component).
		WithActivationConstraint(constraints.Complete(server.GROUP))
}

type Settings struct {
	model *zonemodel.Model
}

type Factory struct {
	support.DefaultFactory[support.None, *Settings, *corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry]
}

func (f *Factory) CreateRequest(r *reconciler.BaseRequest[*corednsv1alpha1.CoreDNSEntry], r2 *support.Reconciler[support.None, *Settings, *corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry]) reconciler.ReconcileRequest[*corednsv1alpha1.CoreDNSEntry] {
	return &Request{r}
}
