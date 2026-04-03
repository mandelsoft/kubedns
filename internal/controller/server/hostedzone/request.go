package hostedzone

import (
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconcile"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconciler"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/common"
	"github.com/mandelsoft/kubedns/internal/controller/server/zonemodel"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO: plain mode support

type Request struct {
	*reconciler.BaseRequest[*corednsv1alpha1.HostedZone]
	model *zonemodel.Model
}

func (r *Request) Reconcile() reconcile.Problem {
	zk := zonemodel.ZoneKeyFromObject(r.ClusterName, r.Object)
	if err := r.Validate(r.Object); err != nil {
		r.model.RemoveZone(zk)
		return reconcile.Failed(err)
	}
	r.Info("update model for {{key}}", "key", zk)
	r.model.AddZone(r.Cluster, zk, r.Object)
	return nil
}

func (r *Request) ReconcileDeleting() reconcile.Problem {
	zk := zonemodel.ZoneKeyFromObject(r.ClusterName, r.Object)
	r.Info("delete {{key}} from model", "key", zk)
	r.model.RemoveZone(zk)
	return nil
}

func (r *Request) ReconcileDeleted() reconcile.Problem {
	zk := zonemodel.NewZoneKey(r.ClusterName, r.Request.Namespace, r.Request.Name)
	r.Info("delete {{key}} from model", "key", zk)
	r.model.RemoveZone(zk)
	return nil
}

func (r *Request) Validate(e *corednsv1alpha1.HostedZone) error {
	reason, err := common.ValidateZone(e)
	if err != nil {
		meta.SetStatusCondition(&e.Status.Conditions, metav1.Condition{
			Type:               corednsv1alpha1.ServerConditionType,
			Status:             metav1.ConditionFalse,
			ObservedGeneration: e.ObjectMeta.Generation,
			Reason:             reason,
			Message:            err.Error(),
		})
	} else {
		meta.SetStatusCondition(&e.Status.Conditions, metav1.Condition{
			Type:               corednsv1alpha1.ServerConditionType,
			Status:             metav1.ConditionTrue,
			ObservedGeneration: e.ObjectMeta.Generation,
			Reason:             corednsv1alpha1.ReasonServerActive,
			Message:            "hosted zone served",
		})
	}
	return err
}
