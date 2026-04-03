package hostedzone

import (
	"github.com/mandelsoft/kubecrtutils/cluster"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Mode interface {
	RuntimeNamespace(c cluster.Cluster, key client.ObjectKey) string
	RuntimeSecretName(c cluster.Cluster, key client.ObjectKey) string
	RuntimeDeploymentName(c cluster.Cluster, key client.ObjectKey) string

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
