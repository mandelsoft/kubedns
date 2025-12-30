package hostedzone

import (
	"fmt"

	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	. "github.com/mandelsoft/kubedns/pkg/controllerutils/reconcile"
	"github.com/mandelsoft/kubedns/pkg/render"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *ReconcileRequest) DeleteExternalResources() Problem {

	values, _ := r.Values(r.reconciler.Mode, true)

	r.Info("rendering manifests to determine objects to be deleted")
	dataplane, runtime, err := render.Render(r.reconciler.Manifests, values)
	if err != nil {
		r.SetStatusCondition(metav1.Condition{
			Type:    corednsv1alpha1.RuntimeConditionType,
			Status:  metav1.ConditionFalse,
			Reason:  corednsv1alpha1.ReasonRuntimeInternalError,
			Message: err.Error(),
		})
		return Failed(err)
	}

	var prob Problem
	r.Info("deleting runtime resources")
	for _, data := range runtime {
		prob = AggregateProblem(prob, r.Delete(r.reconciler.Runtime, data))
	}

	r.Info("deleting dataplane resources")
	for _, data := range dataplane {
		prob = AggregateProblem(prob, r.Delete(r.reconciler.DataPlane, data))
	}

	if prob == nil {
		r.Info("cleanup deployment mode")
		prob = r.reconciler.Mode.Cleanup(r.ReconcileContext, values["dataplane"].(map[string]interface{})["name"].(string))
		if prob != nil {
			err = fmt.Errorf("error cleanup  mode: %s", prob.Message())
		}
	}
	return prob
}
