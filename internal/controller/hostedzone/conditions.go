package hostedzone

import (
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _transferConditions = sets.New[string](
	corednsv1alpha1.RuntimeConditionType,
	corednsv1alpha1.NameserverConditionType,
)

var _myConditions = sets.New[string](
	corednsv1alpha1.ValidationConditionType,
).Insert(_transferConditions.UnsortedList()...)

func (r *ReconcileRequest) transferConditions(src client.ObjectKey, from []metav1.Condition) {
	r.Info("transferring conditions from {{source}}", "source", src)
	for _, c := range from {
		if _transferConditions.Has(c.Type) {
			r.Info("  transferring condition {{condition}}", "condition", c.Type)
			r.SetStatusCondition(metav1.Condition{
				Type:    c.Type,
				Status:  c.Status,
				Reason:  c.Reason,
				Message: c.Message,
			})
		}
	}
	for _, c := range r.instance.Status.Conditions {
		if _myConditions.Has(c.Type) {
			if meta.FindStatusCondition(from, c.Type) == nil {
				r.Info("  removing condition {{condition}}", "condition", c.Type)
				meta.RemoveStatusCondition(&r.instance.Status.Conditions, c.Type)
			}
		}
	}
	r.summary()
}

func (r *ReconcileRequest) summary() {
	status := "Failed"
	msg := "status unknown"
	c := meta.FindStatusCondition(r.instance.Status.Conditions, corednsv1alpha1.ValidationConditionType)
	if c != nil {
		msg = c.Message
		if c.Status == metav1.ConditionTrue {
			c = meta.FindStatusCondition(r.instance.Status.Conditions, corednsv1alpha1.RuntimeConditionType)
			if c != nil {
				msg = c.Message
				if c.Status == metav1.ConditionTrue {
					c = meta.FindStatusCondition(r.instance.Status.Conditions, corednsv1alpha1.NameserverConditionType)
					if c != nil {
						msg = c.Message
						if c.Status == metav1.ConditionTrue {
							c = meta.FindStatusCondition(r.instance.Status.Conditions, corednsv1alpha1.ServerConditionType)
							if c != nil {
								msg = c.Message
								if c.Status == metav1.ConditionTrue {
									status = corednsv1alpha1.STATE_READY
								}
							}
						} else { // NameServer access
							status = "Pending"
						}
					}
				} else { // Runtime
					if c.Reason == corednsv1alpha1.ReasonRuntimeDeploying {
						status = "Pending"
					}
				}
			}
		}
	}
	r.Info("summaried status", "status", status, "message", msg)
	r.instance.Status.Message = msg
	r.instance.Status.State = status
}
