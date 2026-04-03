package entry

import (
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconcile"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconciler"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/common"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO: plain mode support

type Request struct {
	*reconciler.BaseRequest[*corednsv1alpha1.CoreDNSEntry]
}

func (r *Request) Reconcile() reconcile.Problem {
	if err := r.Validate(r.Object); err != nil {
		return reconcile.Failed(err)
	}
	return nil
}

func (r *Request) ReconcileDeleting() reconcile.Problem {
	return nil
}

func (r *Request) ReconcileDeleted() reconcile.Problem {
	return nil
}

func (r *Request) Validate(e *corednsv1alpha1.CoreDNSEntry) error {
	err := common.ValidateEntryData(e)
	if err != nil {
		meta.SetStatusCondition(&e.Status.Conditions, metav1.Condition{
			Type:               corednsv1alpha1.ServerConditionType,
			Status:             metav1.ConditionFalse,
			ObservedGeneration: e.ObjectMeta.Generation,
			Reason:             corednsv1alpha1.ReasonServerValidationFailure,
			Message:            err.Error(),
		})
	} else {
		meta.SetStatusCondition(&e.Status.Conditions, metav1.Condition{
			Type:               corednsv1alpha1.ServerConditionType,
			Status:             metav1.ConditionTrue,
			ObservedGeneration: e.ObjectMeta.Generation,
			Reason:             corednsv1alpha1.ReasonServerActive,
			Message:            "entry served",
		})
	}
	return err
}
