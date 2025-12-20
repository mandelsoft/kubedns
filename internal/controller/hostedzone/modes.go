package hostedzone

import (
	"encoding/base64"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Mode interface {
	RuntimeNamespace(key client.ObjectKey) string
	RuntimeSecretName(key client.ObjectKey) string
	RuntimeDeploymentName(key client.ObjectKey) string

	AccessValues(ctx ReconcileContext, name string) (map[string]interface{}, error)

	Delete(ctx ReconcileContext, name string) error
}

////////////////////////////////////////////////////////////////////////////////

type ModeImpl struct {
	*HostedZoneReconciler
}

////////////////////////////////////////////////////////////////////////////////

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

func (m *LocalMode) AccessValues(ReconcileContext, string) (map[string]interface{}, error) {
	return nil, nil
}

func (r *LocalMode) Delete(ctx ReconcileContext, name string) error {
	return nil
}

////////////////////////////////////////////////////////////////////////////////

type RuntimeMode struct {
	ModeImpl
}

var _ Mode = (*RuntimeMode)(nil)

func NewRuntimeMode(r *HostedZoneReconciler) Mode {
	return &RuntimeMode{ModeImpl{r}}
}

func (m *RuntimeMode) RuntimeNamespace(key client.ObjectKey) string {
	if m.Options.RuntimeNamespace != "" {
		return m.Options.RuntimeNamespace
	}
	return fmt.Sprintf("%s-%s", BASE, key.Namespace)
}

func (m *RuntimeMode) RuntimeSecretName(key client.ObjectKey) string {
	if m.Options.RuntimeNamespace != "" {
		return fmt.Sprintf("%s-%s", BASE, key.Namespace)
	}
	return fmt.Sprintf("%s", BASE)
}

func (m *RuntimeMode) RuntimeDeploymentName(key client.ObjectKey) string {
	if m.Options.RuntimeNamespace != "" {
		return fmt.Sprintf("%s-%s-%s", BASE, key.Namespace, key.Name)
	}
	return fmt.Sprintf("%s-%s", BASE, key.Name)
}

func (m *RuntimeMode) AccessValues(ctx ReconcileContext, name string) (map[string]interface{}, error) {
	var secret v1.Secret

	if ctx.Simulate {
		secret.Data = map[string][]byte{
			"token":  []byte(base64.StdEncoding.EncodeToString([]byte("access-token"))),
			"ca.crt": []byte(base64.StdEncoding.EncodeToString([]byte("server-ca-cert"))),
		}
	} else {
		err := m.DataPlane.Get(ctx, client.ObjectKey{Namespace: ctx.Namespace, Name: name}, &secret)
		if err != nil {
			if errors.IsNotFound(err) {
				secret.Name = name
				secret.Namespace = ctx.Namespace
				secret.Type = v1.SecretTypeServiceAccountToken
				secret.Finalizers = []string{m.Finalizer}
				secret.SetAnnotations(
					map[string]string{v1.ServiceAccountNameKey: name},
				)
				err = m.DataPlane.Create(ctx, &secret)
			}
			if err != nil {
				ctx.Error(err, "cannot get secret", "name", name)
				return nil, err
			}
		}
	}

	token := secret.Data["token"]
	if token == nil {
		return nil, fmt.Errorf("token not yet available")
	}
	access := map[string]interface{}{}
	cert := secret.Data["ca.crt"]
	if cert != nil {
		access["ca.crt"] = string(cert)
	}
	return access, nil
}

func (m *RuntimeMode) Delete(ctx ReconcileContext, name string) error {
	var secret v1.Secret
	err := m.DataPlane.Get(ctx, client.ObjectKey{Namespace: ctx.Namespace, Name: name}, &secret)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if controllerutil.RemoveFinalizer(&secret, m.Finalizer) {
		ctx.Info("removing finalizer from secret")
		if err := m.DataPlane.Update(ctx, &secret); err != nil {
			return err
		}
	}
	if secret.GetDeletionTimestamp().IsZero() {
		ctx.Info("request deletion of secret")
		err = m.DataPlane.Delete(ctx, &secret)
	}
	return err
}
