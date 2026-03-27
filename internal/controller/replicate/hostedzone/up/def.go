package up

import (
	"github.com/mandelsoft/kubecrtutils/controller"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconcile"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/common"
	"github.com/mandelsoft/kubedns/internal/controller/replicate"
	"github.com/mandelsoft/kubedns/internal/controller/replicate/common/up"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Controller() controller.Definition {
	return up.Controller[*corednsv1alpha1.HostedZone, corednsv1alpha1.HostedZone](
		replicate.ControllerHostedzone,
		replicate.HOSTEDZONE_GROUP,
		Handler{},
	)
}

type Handler struct {
}

func (h Handler) SetStatusCondition(obj *corednsv1alpha1.HostedZone, condition metav1.Condition) bool {
	if condition.ObservedGeneration == 0 {
		condition.ObservedGeneration = obj.GetGeneration()
	}
	return meta.SetStatusCondition(&obj.Status.Conditions, condition)
}

func (h Handler) SetResponsibility(r *up.ReconcileRequest[*corednsv1alpha1.HostedZone, corednsv1alpha1.HostedZone], obj *corednsv1alpha1.HostedZone) {
	if obj.Spec.ParentRef == "" {
		if r.Reconciler.Options.TargetClass != "" {
			obj.Spec.Class = &r.Reconciler.Options.TargetClass
		} else {
			obj.Spec.Class = nil
		}
	}
}

func (h Handler) IsResponsible(r *up.ReconcileRequest[*corednsv1alpha1.HostedZone, corednsv1alpha1.HostedZone]) (bool, reconcile.Problem) {
	info, ok, prob := common.GetRootInfo(r, r, r, r.Object)

	if prob != nil {
		if ok {
			// config problem, cannot determine responsibility
			r.Info("cannot determine responsibility", "problem", prob)
			h.SetStatusCondition(r.Object, metav1.Condition{
				Type:    corednsv1alpha1.ValidationConditionType,
				Status:  metav1.ConditionFalse, // Use metav1 constant
				Reason:  corednsv1alpha1.ReasonInvalidParent,
				Message: prob.Error().Error(),
			})
			r.Object.Status.Message = prob.Error().Error()
			r.Object.Status.State = "Problem"
			return false, prob
		}
		// temporary problem, potentially responsible
		r.Info("temporary problem", "problem", prob)
		return true, prob
	}
	if !ok {
		r.Info("oops, unknown state")
		return true, nil
	}

	if info.Class == r.Reconciler.Options.Class {
		return true, nil
	}
	r.Info("not responsible: found class \"{{class}}\", but expected \"{{expected}}\"", "class", info.Class, "expected", r.Reconciler.Options.Class)
	return false, nil
}
