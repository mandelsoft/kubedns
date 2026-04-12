package entry

import (
	"github.com/mandelsoft/kubecrtutils/controller"
	"github.com/mandelsoft/kubecrtutils/controller/constraints"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconciler"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconciler/factories"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/server"
)

func Controller() controller.Definition {
	return controller.Define[*corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry](server.ControllerEntry, server.CLUSTER,
		factories.NewByFactory[*server.Options, factories.None, *corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry](&Factory{}),
	).
		InGroup(server.GROUP).
		UseComponent(server.Component).
		WithActivationConstraint(constraints.Complete(server.GROUP))
}

type Factory struct {
	factories.DefaultFactory[*server.Options, factories.None, *corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry]
}

func (f *Factory) CreateOptions() *server.Options {
	return server.NewOptions()
}

func (f *Factory) CreateRequest(req *reconciler.BaseRequest[*corednsv1alpha1.CoreDNSEntry], r *factories.Reconciler[*server.Options, factories.None, *corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry]) reconciler.ReconcileRequest[*corednsv1alpha1.CoreDNSEntry] {
	if req.Object == nil {
		return &Request{req, r.Options, !r.Options.Slave}
	}
	return &Request{req, r.Options, r.Options.IsMaster(req.Object.Status.Conditions)}
}
