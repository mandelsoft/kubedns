package hostedzone

import (
	"fmt"

	"github.com/mandelsoft/kubedns/pkg/controllerutils/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type LocalMode struct {
	ModeImpl
}

var _ Mode = (*LocalMode)(nil)

func NewLocalMode(r *HostedZoneReconciler) Mode {
	return &LocalMode{ModeImpl{r}}
}

func (m *LocalMode) RuntimeNamespace(key client.ObjectKey) string {
	return key.Namespace
}

func (m *LocalMode) RuntimeSecretName(client.ObjectKey) string {
	return ""
}

func (m *LocalMode) RuntimeDeploymentName(key client.ObjectKey) string {
	return fmt.Sprintf("%s-%s", BASE, key.Name)
}

func (m *LocalMode) AccessValues(ctx ReconcileContext, name string, deleting bool) (map[string]interface{}, error) {
	key := client.ObjectKey{Namespace: ctx.Namespace, Name: name}
	if !ctx.Simulate {
		m.index.Remove(INDEX_SASECFRET, ctx.ObjectKey, key)
	}
	return nil, nil
}

func (m *LocalMode) Prepare(ctx ReconcileContext) reconcile.Problem {
	return nil
}
