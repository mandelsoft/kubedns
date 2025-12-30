package hostedzone

import (
	"encoding/base64"
	"fmt"

	. "github.com/mandelsoft/kubedns/pkg/controllerutils/reconcile"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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

func (m *RuntimeMode) AccessValues(ctx ReconcileContext, name string, deleting bool) (map[string]interface{}, error) {
	var secret v1.Secret

	if ctx.Simulate {
		secret.Data = map[string][]byte{
			"token":  []byte(base64.StdEncoding.EncodeToString([]byte("access-token"))),
			"ca.crt": []byte(base64.StdEncoding.EncodeToString([]byte("server-ca-cert"))),
		}
	} else {
		key := client.ObjectKey{Namespace: ctx.Namespace, Name: name}
		if !ctx.Simulate {
			m.index.Add(INDEX_SASECFRET, ctx.ObjectKey, key)
		}
		err := m.DataPlane.Get(ctx, key, &secret)
		if err != nil {
			if errors.IsNotFound(err) {
				if deleting {
					ctx.Info("serviceaccount secret {{secret}} already gone", "secret", key)
					return nil, nil
				} else {
					ctx.Info("creating serviceaccount secret {{secret}}", "secret", key)
					secret.Name = name
					secret.Namespace = ctx.Namespace
					secret.Type = v1.SecretTypeServiceAccountToken
					// secret.Finalizers = []string{m.Finalizer}
					// don't use finalizers. SA secrets are instantly deleted after creation if there
					// is no matching sa. If the finalizer is set it will stuck in deletion and
					// it will never be populated.
					secret.SetAnnotations(
						map[string]string{
							v1.ServiceAccountNameKey:   name,
							"mandelsoft.org/generated": GENERATED,
						},
					)
					err = m.DataPlane.Create(ctx, &secret)
				}
			}
			if err != nil {
				ctx.LogError(err, "cannot get secret", "name", name)
				return nil, err
			}
		}
	}

	token := secret.Data["token"]
	if token == nil {
		return nil, fmt.Errorf("token not yet available")
	}
	ctx.Info("token found for serviceaccount", "name", name)
	access := map[string]interface{}{}
	access["token"] = string(token)
	cert := secret.Data["ca.crt"]
	if cert != nil {
		access["cadata"] = string(cert)
	}
	return access, nil
}

func (m *RuntimeMode) Prepare(ctx ReconcileContext) Problem {
	// assure target namespace
	var ns v1.Namespace
	namespace := m.RuntimeNamespace(ctx.ObjectKey)
	err := m.Runtime.Get(ctx, client.ObjectKey{Name: namespace}, &ns)
	if err != nil {
		if errors.IsNotFound(err) {
			ctx.Info("assure target namespace", "namespace", namespace)
			ns.Name = namespace
			err = m.Runtime.Create(ctx, &ns)
		}
	}
	return TemporaryProblem(err)
}

func (m *RuntimeMode) Cleanup(ctx ReconcileContext, name string) Problem {
	if prob := m.ModeImpl.Cleanup(ctx, name); prob != nil {
		return prob
	}

	key := client.ObjectKey{Namespace: ctx.Namespace, Name: name}
	if len(m.index.UsersFor(INDEX_SASECFRET, key)) != 0 {
		return nil
	}

	var secret v1.Secret
	if err := m.DataPlane.Get(ctx, key, &secret); err != nil {
		if !errors.IsNotFound(err) {
			return TemporaryProblem(err)
		}
		ctx.Info("serviceaccount secret {{secret}} already gone", "secret", key)
	} else {
		if secret.GetDeletionTimestamp().IsZero() {
			ctx.Info("request deletion of serviceaccount secret {{secret}}", "secret", key)
			err = m.DataPlane.Delete(ctx, &secret)
			if err != nil {
				if !errors.IsNotFound(err) {
					return TemporaryProblem(err)
				}
				ctx.Info(" serviceaccount secret {{secret}} already gone", "secret", key)
			}
		} else {
			ctx.Info(" serviceaccount secret {{secret}} is waiting for finalizers {{finalizers}}", "secret", key, "finalizers", secret.Finalizers)
			return Requeuef("waiting for secret finalizers to be removed")
		}
	}

	if m.Options.RuntimeNamespace == "" {
		var ns v1.Namespace
		namespace := m.RuntimeNamespace(ctx.ObjectKey)
		err := m.Runtime.Get(ctx, client.ObjectKey{Name: namespace}, &ns)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return TemporaryProblem(err)
		}
		if len(ns.Finalizers) > 0 {
			return Requeuef("waiting for namespace finalizers to be removed")
		}

	}
	ctx.Info("cleanup successful")
	return nil
}
