package hostedzone

import (
	"encoding/base64"
	"fmt"

	"github.com/mandelsoft/kubecrtutils/cluster"
	. "github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconcile"
	"github.com/mandelsoft/kubecrtutils/objutils"
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

func (m *RuntimeMode) RuntimeNamespace(c cluster.Cluster, key client.ObjectKey) string {
	if m.Options.RuntimeNamespace != "" {
		return m.Options.RuntimeNamespace
	}
	return objutils.GenerateUniqueName(BASE, c.GetId(), "", key.Namespace, objutils.MAX_NAMESPACELEN)
}

func (m *RuntimeMode) RuntimeSecretName(c cluster.Cluster, key client.ObjectKey) string {
	if m.Options.RuntimeNamespace != "" {
		return objutils.GenerateUniqueName(BASE, c.GetId(), "", key.Namespace, objutils.MAX_NAMESPACELEN)
	}
	return fmt.Sprintf("%s", BASE)
}

func (m *RuntimeMode) RuntimeDeploymentName(c cluster.Cluster, key client.ObjectKey) string {
	if m.Options.RuntimeNamespace != "" {
		return objutils.GenerateUniqueName(BASE, c.GetId(), key.Namespace, key.Name, objutils.MAX_NAMELEN)
	}
	return objutils.GenerateUniqueName(BASE, c.GetId(), "", key.Name, objutils.MAX_NAMELEN)
}

func (m *RuntimeMode) AccessValues(ctx ReconcileContext, name string, deleting bool) (map[string]interface{}, error) {
	var secret v1.Secret

	if ctx.IsSimulate() {
		secret.Data = map[string][]byte{
			"token":  []byte(base64.StdEncoding.EncodeToString([]byte("access-token"))),
			"ca.crt": []byte(base64.StdEncoding.EncodeToString([]byte("server-ca-cert"))),
		}
	} else {
		key := client.ObjectKey{Namespace: ctx.GetKey().Namespace, Name: name}
		if !ctx.IsSimulate() {
			m.index.Add(INDEX_SASECFRET, ctx.GetKey(), key)
		}
		err := ctx.Get(ctx, key, &secret)
		if err != nil {
			if errors.IsNotFound(err) {
				if deleting {
					ctx.Info("serviceaccount secret {{secret}} already gone", "secret", key)
					return nil, nil
				} else {
					ctx.Info("creating serviceaccount secret {{secret}}", "secret", key)
					secret.Name = name
					secret.Namespace = ctx.GetKey().Namespace
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
					err = ctx.Create(ctx, &secret)
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
	namespace := m.RuntimeNamespace(ctx, ctx.GetKey())
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

	key := client.ObjectKey{Namespace: ctx.GetKey().Namespace, Name: name}
	found := len(m.index.UsersFor(INDEX_SASECFRET, key))
	if found != 0 {
		m.Info("found still {{amount}} zones", "amount", found)
		return nil
	}

	var secret v1.Secret
	if err := ctx.Get(ctx, key, &secret); err != nil {
		if !errors.IsNotFound(err) {
			return TemporaryProblem(err)
		}
		ctx.Info("serviceaccount secret {{secret}} already gone", "secret", key)
	} else {
		if secret.GetDeletionTimestamp().IsZero() {
			ctx.Info("request deletion of serviceaccount secret {{secret}}", "secret", key)
			err = ctx.Delete(ctx, &secret)
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
		namespace := m.RuntimeNamespace(ctx, ctx.GetKey())
		m.Info("deleting namespace {{namespace}}", "namespace", namespace)
		err := m.Runtime.Get(ctx, client.ObjectKey{Name: namespace}, &ns)
		if err != nil {
			if !errors.IsNotFound(err) {
				return TemporaryProblem(err)
			}
			m.Info("namespace {{namespace}} already deleted", "namespace", namespace)
		} else {
			if ns.DeletionTimestamp.IsZero() {
				err := m.Runtime.Delete(ctx, &ns)
				if err != nil {
					if !errors.IsNotFound(err) {
						return TemporaryProblem(err)
					}
					m.Info("namespace {{namespace}} already deleted", "namespace", namespace)
				}
			} else {
				m.Info("namespace {{namespace}} still deleting", "namespace", namespace)
			}
		}
		if len(ns.Finalizers) > 0 {
			return Requeuef("waiting for namespace finalizers to be removed")
		}

	}
	ctx.Info("cleanup completed")
	return nil
}
