package hostedzone

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Mode interface {
	RuntimeNamespace(key client.ObjectKey) string
	RuntimeSecretName(key client.ObjectKey) string
	RuntimeDeploymentName(key client.ObjectKey) string

	AccessValues(ctx ReconcileContext, name string, deleting bool) (map[string]interface{}, error)

	Prepare(ctx ReconcileContext) error
	Cleanup(ctx ReconcileContext, name string) error
}

////////////////////////////////////////////////////////////////////////////////

type ModeImpl struct {
	*HostedZoneReconciler
}

func (m *ModeImpl) Cleanup(ctx ReconcileContext, name string) error {
	key := client.ObjectKey{Namespace: ctx.Namespace, Name: name}
	if !ctx.Simulate {
		m.index.Remove(INDEX_SASECFRET, ctx.ObjectKey, key)
	}
	return nil
}
