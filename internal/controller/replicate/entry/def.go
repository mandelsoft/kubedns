package entry

import (
	"github.com/mandelsoft/kubecrtutils/controller"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconcile"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/common"
	"github.com/mandelsoft/kubedns/internal/controller/replicate"
	"github.com/mandelsoft/kubedns/internal/controller/replicate/common/generic"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Controller() controller.Definition {
	return generic.Controller[*corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry](
		replicate.ControllerEntry,
		replicate.GROUP,
		Responsibility,
	)
}

func Responsibility(c controller.TypedController[*corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry]) (generic.ResponsibilityHandler[*corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry], error) {
	return &Handler{}, nil
}

type Handler struct {
}

func (h *Handler) Delete(r generic.Request[*corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry]) {
}

func (h *Handler) SetResponsibility(r generic.Request[*corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry], obj *corednsv1alpha1.CoreDNSEntry) {
}

func (h *Handler) IsResponsible(r generic.Request[*corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry]) (bool, reconcile.Problem) {
	info, err, prob := common.ValidateEntry(r, r, r, r.Object)

	if prob != nil {
		return true, prob
	}

	if info != nil {
		if info.Class != r.Reconciler.Options.Class {
			r.Info("not responsible: found class \"{{class}}\", but expected \"{{expected}}\"", "class", info.Class, "expected", r.Reconciler.Options.Class)
			return false, nil
		}
	}

	if err != nil {
		if info == nil {
			// config problem, cannot determine responsibility
			r.Info("cannot determine responsibility", "problem", prob)

			r.SetStatusCondition(r.Object, metav1.Condition{
				Type:    corednsv1alpha1.ValidationConditionType,
				Status:  metav1.ConditionFalse,
				Reason:  corednsv1alpha1.ReasonConfigurarationInvalid,
				Message: err.Error(),
			})
			r.Object.Status.State = "Problem"
			r.Object.Status.Message = err.Error()
			return false, reconcile.Failed(err)
		} else {
			r.Info("configuration problem propagated up stream: {{problem}}", "problem", err)
		}
	}
	return true, nil
}
