package up

import (
	"github.com/mandelsoft/kubecrtutils"
	"github.com/mandelsoft/kubecrtutils/cluster"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconcile"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconciler"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconciler/factories"
	"github.com/mandelsoft/kubecrtutils/objutils"
	"github.com/mandelsoft/kubedns/internal/controller/replicate"
	"github.com/mandelsoft/kubedns/internal/controller/replicate/common"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Settings[P kubecrtutils.ObjectPointer[T], T any] struct {
	Target  cluster.Cluster
	Mapping common.Mapping
	Resp    ResponsibilityHandler[P, T]
}

type ReconcileRequest[P kubecrtutils.ObjectPointer[T], T any] struct {
	reconciler.DefaultReconcileRequest[P, *factories.Reconciler[*common.Options, Settings[P, T], P, T]]
	MappingContext common.Context
}

func (r *ReconcileRequest[P, T]) Reconcile() reconcile.Problem {
	if objutils.GetAnnotation(r.Object, replicate.ANNOTATION) != "" {
		r.Info("skip replicated object")
		return nil
	}
	s := r.Reconciler.Settings

	if s.Resp != nil {
		ok, prob := s.Resp.IsResponsible(r)

		if !ok {
			r.Info("handle replica deletion for being not reponsible")
			p := r.ReconcileDeleting()
			if p != nil {
				return p
			}
			return prob
		}
		if prob != nil {
			return prob
		}
	}
	patch := client.MergeFrom(r.GetOrig())
	if controllerutil.AddFinalizer(r.Object, r.Reconciler.Finalizer) {
		if err := r.Patch(r, r.Object, patch); err != nil {
			return reconcile.TemporaryProblem(client.IgnoreNotFound(err))
		}
		r.Info("taking responsibility")
	}

	// assure target namespace
	key := MapKey(r.NamespacedName, r, r.Reconciler.Options.TargetNamespace)
	prob := s.Mapping.SetOriginal(r.MappingContext, key, r.Request)
	if prob != nil {
		return prob
	}

	// update replica
	newp := r.Object.DeepCopyObject().(P)
	newp.SetNamespace(key.Namespace)
	newp.SetName(key.Name)
	objutils.CleanupMeta(newp)
	objutils.SetAnnotation(newp, replicate.ANNOTATION, r.Cluster.GetId())
	controllerutil.RemoveFinalizer(newp, r.Reconciler.Finalizer)
	if s.Resp != nil {
		s.Resp.SetResponsibility(r, newp)
	}
	err := r.Reconciler.Options.OwnerHandler.SetOwner(r.Cluster, r.Object, s.Target, newp)
	if err != nil {
		return reconcile.TemporaryProblem(err)
	}
	var tgt T
	tgtp := P(&tgt)
	err = s.Target.Get(r.Context, key, tgtp)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.TemporaryProblem(err)
		}
		r.Info("create in target")
		err = s.Target.Create(r.Context, newp, &client.CreateOptions{
			FieldManager: r.Reconciler.FieldManager,
		})
	} else {
		if tgtp.GetDeletionTimestamp() != nil {
			// complete deletion before recreation
			r.Info("replica is deleting -> wait to be completed")
			return nil
		}
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
		_, err = cluster.ClientSideApplyObject(s.Target, cluster.DefaultOperationContext(r, r, ""), newp, tgtp)
	}
	return reconcile.TemporaryProblem(client.IgnoreNotFound(err))
}

func (r *ReconcileRequest[P, T]) ReconcileDeleting() reconcile.Problem {
	key := MapKey(r.NamespacedName, r, r.Reconciler.Options.TargetNamespace)
	s := r.Reconciler.Settings

	var tgt T
	tgtp := P(&tgt)
	err := s.Target.Get(r.Context, key, tgtp)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.TemporaryProblem(err)
		}
		r.Info("replica already deleted")
		patch := client.MergeFrom(r.GetOrig())
		if controllerutil.RemoveFinalizer(r.Object, r.Reconciler.Finalizer) {
			r.Info("releasing responsibility")
			if err := r.Patch(r, r.Object, patch); err != nil {
				return reconcile.TemporaryProblem(client.IgnoreNotFound(err))
			}
		}
		return s.Mapping.RemoveOriginal(r.MappingContext, key)
	}
	patch := client.MergeFrom(tgtp.DeepCopyObject().(client.Object))
	if controllerutil.RemoveFinalizer(tgtp, r.Reconciler.Finalizer) {
		r.Info("removing finalizer from replica")
		if err := r.Patch(r, tgtp, patch); err != nil {
			return reconcile.TemporaryProblem(client.IgnoreNotFound(err))
		}
	}

	if s.Resp != nil {
		defer s.Resp.Delete(r)
	}
	if tgtp.GetDeletionTimestamp() != nil {
		r.Info("replica already deleting")
	} else {
		r.Info("request replica deletion")
		return reconcile.TemporaryProblem(s.Target.Delete(r.Context, tgtp))
	}
	return nil
}

func MapKey(key client.ObjectKey, c cluster.Cluster, tgtns string) client.ObjectKey {
	if tgtns != "" {
		name := objutils.GenerateUniqueName("replica", c.GetId(), key.Name, key.Namespace, objutils.MAX_NAMELEN)
		return client.ObjectKey{
			Name:      name,
			Namespace: tgtns,
		}
	}
	namespace := objutils.GenerateUniqueName("replica", c.GetId(), "", key.Namespace, objutils.MAX_NAMESPACELEN)
	return client.ObjectKey{
		Name:      key.Name,
		Namespace: namespace,
	}

}
