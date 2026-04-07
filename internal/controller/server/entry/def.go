package entry

import (
	"github.com/mandelsoft/kubecrtutils/controller"
	"github.com/mandelsoft/kubecrtutils/controller/constraints"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconciler"
	"github.com/mandelsoft/kubecrtutils/controller/support"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/server"
)

func Controller() controller.Definition {
	return controller.Define[*corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry](server.ControllerEntry, server.CLUSTER,
		support.NewByFactory[*server.Options, support.None, *corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry](&Factory{}),
	).
		InGroup(server.GROUP).
		UseComponent(server.Component).
		WithActivationConstraint(constraints.Complete(server.GROUP))
}

type Factory struct {
	support.DefaultFactory[*server.Options, support.None, *corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry]
}

func (f *Factory) CreateOptions() *server.Options {
	return server.NewOptions()
}

func (f *Factory) CreateRequest(r *reconciler.BaseRequest[*corednsv1alpha1.CoreDNSEntry], r2 *support.Reconciler[*server.Options, support.None, *corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry]) reconciler.ReconcileRequest[*corednsv1alpha1.CoreDNSEntry] {
	if r.Object == nil {
		return &Request{r, r2.Options, !r2.Options.Slave}
	}
	return &Request{r, r2.Options, r2.Options.IsMaster(r.Object.Status.Conditions)}
}
