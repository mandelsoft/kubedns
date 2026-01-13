package hostedzone

import (
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/controller/controllerutils/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Mode interface {
	RuntimeNamespace(key client.ObjectKey) string
	RuntimeSecretName(key client.ObjectKey) string
	RuntimeDeploymentName(key client.ObjectKey) string

	AccessValues(ctx ReconcileContext, name string, deleting bool) (map[string]interface{}, error)

	Prepare(ctx ReconcileContext) reconcile.Problem
	Cleanup(ctx ReconcileContext, name string) reconcile.Problem
}

////////////////////////////////////////////////////////////////////////////////

type ModeImpl struct {
	*HostedZoneReconciler
}

func (m *ModeImpl) Cleanup(ctx ReconcileContext, name string) reconcile.Problem {
	secretkey := client.ObjectKey{Namespace: ctx.GetKey().Namespace, Name: name}
	if !ctx.IsSimulate() {
		m.index.Remove(INDEX_SASECFRET, ctx.GetKey(), secretkey)
	}
	return nil
}
