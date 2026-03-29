package down

import (
	"github.com/mandelsoft/kubecrtutils"
	"github.com/mandelsoft/kubecrtutils/cluster"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconcile"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconciler"
	"github.com/mandelsoft/kubecrtutils/controller/support"
	"github.com/mandelsoft/kubecrtutils/objutils"
	"github.com/mandelsoft/kubecrtutils/owner"
	"github.com/mandelsoft/kubedns/internal/controller/replicate"
	"github.com/mandelsoft/kubedns/internal/controller/replicate/common"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Settings struct {
	Source  cluster.ClusterEquivalent
	Mapping common.Mapping
}

type ReconcileRequest[P kubecrtutils.ObjectPointer[T], T any] struct {
	MappingContext common.Context
	reconciler.DefaultReconcileRequest[P, *support.Reconciler[*common.Options, Settings, P, T]]
}

func (r *ReconcileRequest[P, T]) Reconcile() reconcile.Problem {
	if objutils.GetAnnotation(r.Object, replicate.ANNOTATION) == "" {
		r.Info("skip non-replicated object")
		return nil
	}

	s := r.Reconciler

	cname, key := s.Options.OwnerHandler.GetOwner(owner.MatcherFor(s.Settings.Source), r.Cluster, r.Object, s.GroupKind)
	if key == nil {
		r.Info("missing owner")
		return nil
	}
	r.Info("found owner {{owner}} in {{cluster}}", "owner", key, "cluster", cname)

	c := cluster.GetClusterFor(r.Cluster, cname)
	if c == nil {
		r.Info("cluster {{cluster}} not found", "cluster", cname)
		return nil
	}

	var orig T
	err := c.Get(r, *key, P(&orig))
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.TemporaryProblem(err)
		}
		r.Info("original object not found -> delete replica")
		err = r.Cluster.AsCluster().Delete(r, r.Object)
		if !errors.IsNotFound(err) {
			return reconcile.TemporaryProblem(err)
		}
		return nil
	}
	r.Info("transfer status to {{orig}} in {{source}}", "orig", key, "source", cname)
	patch := client.MergeFrom(P(&orig).DeepCopyObject().(client.Object))

	status, err := objutils.GetStatusField(r.Object)
	if err != nil {
		r.Info("cannot determine status field")
		return nil
	}
	err = objutils.SetStatusField(P(&orig), status)
	if err != nil {
		r.Info("cannot update status field")
		return nil
	}
	return reconcile.TemporaryProblem(c.GetClient().Status().Patch(r, P(&orig), patch))
}

func (r *ReconcileRequest[P, T]) ReconcileDeleted() reconcile.Problem {
	key := r.Reconciler.Settings.Mapping.GetOriginal(r.Request.Request.NamespacedName)
	if key == nil {
		return nil
	}

	s := r.Reconciler.Settings

	r.Info("found source {{source}} for deleted object", "source", *key)

	var orig T
	origp := P(&orig)

	c := cluster.GetClusterFor(s.Source, key.ClusterName)
	if c == nil {
		return s.Mapping.RemoveOriginal(r.MappingContext, r.Request.Request.NamespacedName)
	}

	err := c.Get(r, key.NamespacedName, origp)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Info("original object already gone")
			return s.Mapping.RemoveOriginal(r, r.Request.Request.NamespacedName)
		}
		return reconcile.TemporaryProblem(err)
	}
	if origp.GetDeletionTimestamp() != nil {
		patch := client.MergeFrom(origp.DeepCopyObject().(client.Object))
		if controllerutil.RemoveFinalizer(origp, r.Reconciler.Finalizer) {
			r.Info("original still deleting -> remove finalizer")
			if err := r.Patch(r, origp, patch); err != nil {
				return reconcile.TemporaryProblem(client.IgnoreNotFound(err))
			}
		} else {
			r.Info("original still deleting")
		}
		return s.Mapping.RemoveOriginal(r, r.Request.Request.NamespacedName)
	}

	r.Info("original object still valid -> trigger recreation")
	err = c.EnqueueByObject(r, origp)
	if err != nil {
		r.Error("cannot enqueue {{key}}", key)
	}
	return s.Mapping.RemoveOriginal(r, r.Request.Request.NamespacedName)
}
