package hostedzone

import (
	"fmt"

	"github.com/mandelsoft/goutils/general"
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
	settings *Settings
	master   bool
}

func (r *Request) Reconcile() reconcile.Problem {
	zk := zonemodel.ZoneKeyFromObject(r.Cluster.GetId(), r.Object)

	info, ok, prob := common.GetRootInfo(r, r, r, r.Object, nil)
	if prob != nil {
		if !ok {
			// temp problem
			return prob
		}
		// config problem
		r.settings.model.RemoveZone(zk)
		return r.updateStatus(false, fmt.Errorf("%s", prob.Message()))
	}

	if !info.Check(r.settings.Class, nil) {
		// not responsible
		if r.master {
			if r.Object.Status.Observed != nil && r.Object.Status.Observed.Class == String(r.settings.Class) {
				r.Info("lost responsibility")
				r.Object.Status.Observed = nil
			}
		}
		r.settings.model.RemoveZone(zk)
		return nil
	}

	if r.master {
		r.Object.Status.Observed = &corednsv1alpha1.Observed{
			Class:   String(r.settings.Class),
			Runtime: String(r.Object.Spec.Runtime),
		}
	}
	if reason, err := common.ValidateZone(r.Object); err != nil {
		r.settings.model.RemoveZone(zk)
		return r.updateStatus(true, err, reason)
	}
	r.Info("update model for {{key}}", "key", zk)
	r.settings.model.AddZone(r.Cluster, zk, r.Object)
	return r.updateStatus(true, nil)
}

func (r *Request) ReconcileDeleting() reconcile.Problem {
	zk := zonemodel.ZoneKeyFromObject(r.ClusterName, r.Object)
	r.Info("delete {{key}} from model", "key", zk)
	r.settings.model.RemoveZone(zk)
	r.TriggerStatusChanged()
	return nil
}

func (r *Request) ReconcileDeleted() reconcile.Problem {
	zk := zonemodel.NewZoneKey(r.Cluster.GetId(), r.Request.Namespace, r.Request.Name)
	r.Info("delete {{key}} from model", "key", zk)
	r.settings.model.RemoveZone(zk)
	r.TriggerStatusChanged()
	return nil
}

func (r *Request) TriggerStatusChanged() {
	// update entries for deleted zone
	r.settings.TriggerChildren(r, r, r.Request.NamespacedName)
	r.settings.TriggerEntries(r, r, r.Request.NamespacedName)
}

func (r *Request) updateStatus(force bool, err error, oreason ...string) reconcile.Problem {
	o := r.Object
	if r.master {
		if len(o.Status.Conditions) > 0 {
			o.Status.Conditions = nil
		}
	}

	// In plain mode the status is managed directly. otherwise
	// a server condition is managed, which is the handled by the aaS controller
	// to determine an aggregated object state.
	if err != nil {
		reason := general.OptionalDefaulted(corednsv1alpha1.ReasonServerValidationFailure, oreason...)

		if r.master {
			if force || o.Status.State != "Invalid" {
				if o.Status.Message != err.Error() || o.Status.State != "Invalid" {
					o.Status.Message = err.Error()
					o.Status.State = "Invalid"
				}
			}
		} else {
			meta.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
				Type:               corednsv1alpha1.ServerConditionType,
				Status:             metav1.ConditionFalse,
				ObservedGeneration: o.ObjectMeta.Generation,
				Reason:             reason,
				Message:            err.Error(),
			})
		}
	} else {
		if r.master {
			if o.Status.Message != "zone is served" || o.Status.State != "Ok" {
				o.Status.Message = "zone is served"
				o.Status.State = "Ok"
			}
		} else {
			meta.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
				Type:               corednsv1alpha1.ServerConditionType,
				Status:             metav1.ConditionTrue,
				ObservedGeneration: o.ObjectMeta.Generation,
				Reason:             corednsv1alpha1.ReasonServerActive,
				Message:            "hosted zone served",
			})
		}
	}
	return nil
}

func String(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
