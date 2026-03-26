package up

import (
	"github.com/mandelsoft/kubecrtutils"
	"github.com/mandelsoft/kubecrtutils/cluster"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconcile"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconciler"
	"github.com/mandelsoft/kubecrtutils/controller/support"
	"github.com/mandelsoft/kubecrtutils/objutils"
	"github.com/mandelsoft/kubedns/internal/controller/replicate"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Settings struct {
	Target cluster.Cluster
}

type ReconcileRequest[P kubecrtutils.ObjectPointer[T], T any] struct {
	reconciler.DefaultReconcileRequest[P, *support.Reconciler[*replicate.Options, Settings, P, T]]
}

func (r *ReconcileRequest[P, T]) Reconcile() reconcile.Problem {
	if objutils.GetAnnotation(r.Object, replicate.ANNOTATION) != "" {
		r.Info("skip replicated object")
		return nil
	}

	s := r.Reconciler

	// Todo: determine zone to check class

	patch := client.MergeFrom(r.GetOrig())
	if controllerutil.AddFinalizer(r.Object, s.Finalizer) {
		if err := r.Patch(r, r.Object, patch); err != nil {
			return reconcile.TemporaryProblem(client.IgnoreNotFound(err))
		}
		r.Info("taking responsibility")
	}

	// assure target namespace
	namespace := objutils.GenerateUniqueName("replica", r.Cluster.GetId(), "", r.Namespace, objutils.MAX_NAMESPACELEN)
	key := client.ObjectKey{
		Name:      r.Name,
		Namespace: namespace,
	}
	s.Options.SetOriginal(key, r.Request)

	var ns v1.Namespace
	err := s.Settings.Target.Get(r, client.ObjectKey{Name: namespace}, &ns)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Info("assure target namespace", "namespace", namespace)
			ns.Name = namespace
			err = s.Settings.Target.Create(r, &ns, &client.CreateOptions{
				FieldManager: s.FieldManager,
			})
		}
		if err != nil {
			return reconcile.TemporaryProblem(err)
		}
	}

	newp := r.Object.DeepCopyObject().(P)
	newp.SetNamespace(namespace)
	objutils.CleanupMeta(newp)
	objutils.SetAnnotation(newp, replicate.ANNOTATION, r.Cluster.GetId())
	controllerutil.RemoveFinalizer(newp, s.Finalizer)

	r.Reconciler.Options.OwnerHandler.SetOwner(r.Cluster, r.Object, s.Settings.Target, newp)
	var tgt T
	tgtp := P(&tgt)
	err = s.Settings.Target.Get(r.Context, key, tgtp)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.TemporaryProblem(err)
		}
		r.Info("create in target")
		err = s.Settings.Target.Create(r.Context, newp, &client.CreateOptions{
			FieldManager: s.FieldManager,
		})
	} else {
		newp.SetFinalizers(tgtp.GetFinalizers())
		status, err := objutils.GetStatusField(tgtp)
		if err != nil {
			r.Info("cannot determine status field")
			return nil
		}
		err = objutils.SetStatusField(newp, status)
		if err != nil {
			r.Info("cannot set status field")
			return nil
		}

		// pass s.FieldManager to patch only managed fields
		_, err = cluster.ClientSideApplyObject(s.Settings.Target, cluster.DefaultOperationContext(r, r, ""), newp, tgtp)
	}
	return reconcile.TemporaryProblem(client.IgnoreNotFound(err))
}

func (r *ReconcileRequest[P, T]) ReconcileDeleting() reconcile.Problem {
	namespace := objutils.GenerateUniqueName("replica", r.Cluster.GetId(), r.Reconciler.Options.TargetNamespace, r.Namespace, objutils.MAX_NAMESPACELEN)
	s := r.Reconciler

	var tgt T
	tgtp := P(&tgt)
	err := s.Settings.Target.Get(r.Context, client.ObjectKey{Name: r.Name, Namespace: namespace}, tgtp)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.TemporaryProblem(err)
		}
		r.Info("replica already deleted")
		patch := client.MergeFrom(r.GetOrig())
		if controllerutil.RemoveFinalizer(r.Object, s.Finalizer) {
			r.Info("releasing responsibility")
			if err := r.Patch(r, r.Object, patch); err != nil {
				return reconcile.TemporaryProblem(client.IgnoreNotFound(err))
			}
		}
		return nil
	}
	patch := client.MergeFrom(tgtp.DeepCopyObject().(client.Object))
	if controllerutil.RemoveFinalizer(tgtp, s.Finalizer) {
		r.Info("removing finalizer from replica")
		if err := r.Patch(r, tgtp, patch); err != nil {
			return reconcile.TemporaryProblem(client.IgnoreNotFound(err))
		}
	}
	if tgtp.GetDeletionTimestamp() != nil {
		r.Info("replica already deleting")
	} else {
		r.Info("request replica deletion")
		return reconcile.TemporaryProblem(s.Settings.Target.Delete(r.Context, tgtp))
	}
	return nil
}
