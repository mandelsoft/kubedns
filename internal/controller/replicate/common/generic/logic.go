package generic

import (
	"github.com/mandelsoft/kubecrtutils"
	"github.com/mandelsoft/kubecrtutils/cluster"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconcile"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconciler/logic"
	"github.com/mandelsoft/kubecrtutils/controller/replication"
	"github.com/mandelsoft/kubecrtutils/objutils"
	"github.com/mandelsoft/kubedns/internal/controller/replicate"
	"github.com/mandelsoft/kubedns/internal/controller/replicate/common"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Request[P kubecrtutils.ObjectPointer[T], T any] = *logic.Request[*common.Options, Settings[P, T], P, T]

type Settings[P kubecrtutils.ObjectPointer[T], T any] struct {
	Target  cluster.Cluster
	Resp    ResponsibilityHandler[P, T]
	Mapping replication.ResourceMapping
	*common.Options
}

func (f *ReconcilationLogic[P, T]) Reconcile(r Request[P, T]) reconcile.Problem {
	obj := r.GetObject()

	if objutils.GetAnnotation(obj, replicate.REPLICATED_ANNOTATION) != "" {
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

	// taking responsibility
	patch := client.MergeFrom(r.GetOrig())
	if controllerutil.AddFinalizer(r.Object, r.Reconciler.Finalizer) {
		if err := r.Patch(r, r.Object, patch); err != nil {
			return reconcile.TemporaryProblem(client.IgnoreNotFound(err))
		}
		r.Info("taking responsibility")
	}

	// assure target namespace
	key := MapKey(r.NamespacedName, r, s.TargetNamespace)
	prob := s.Mapping.SetOriginal(replication.WithCluster(r, s.Target), key, r.Request)
	if prob != nil {
		return prob
	}

	// update replica
	newp := r.Object.DeepCopyObject().(P)
	newp.SetNamespace(key.Namespace)
	newp.SetName(key.Name)
	objutils.CleanupMeta(newp)
	objutils.SetAnnotation(newp, replicate.REPLICATED_ANNOTATION, r.Cluster.GetId())
	controllerutil.RemoveFinalizer(newp, r.Reconciler.Finalizer)
	if s.Resp != nil {
		s.Resp.SetResponsibility(r, newp)
	}
	err := r.SetOwner(r.Cluster, r.Object, s.Target, newp)
	if err != nil {
		return reconcile.TemporaryProblem(err)
	}

	// update/create replica
	var tgt T
	tgtp := P(&tgt)
	err = s.Target.Get(r, key, tgtp)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.TemporaryProblem(err)
		}
		r.Info("create in target")
		err = s.Target.Create(r, newp, &client.CreateOptions{
			FieldManager: r.Reconciler.FieldManager,
		})
	} else {
		if tgtp.GetDeletionTimestamp() != nil {
			// complete deletion before recreation
			r.Info("replica is deleting -> wait to be completed")
			return nil
		}
		newp.SetFinalizers(tgtp.GetFinalizers())

		// update status
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

func (f *ReconcilationLogic[P, T]) ReconcileDeleting(r Request[P, T]) reconcile.Problem {
	key := MapKey(r.NamespacedName, r, r.Reconciler.Options.TargetNamespace)
	s := r.Reconciler.Settings

	var tgt T
	tgtp := P(&tgt)
	err := s.Target.Get(r, key, tgtp)
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
		return s.Mapping.RemoveOriginal(replication.WithCluster(r, s.Target), key)
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
	return reconcile.Succeeded()
}

func (f *ReconcilationLogic[P, T]) ReconcileDeleted(r Request[P, T]) reconcile.Problem {
	return reconcile.Succeeded()
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
