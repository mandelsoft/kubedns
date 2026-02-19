package hostedzone

import (
	. "github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconcile"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/pkg/render"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *ReconcileRequest) DeleteExternalResources() Problem {

	values, _ := r.Values(r.Reconciler.Mode, true)
	r.Info("rendering manifests to determine objects to be deleted")
	dataplane, runtime, err := render.Render(r.Reconciler.Manifests, values)
	if err != nil {
		r.SetStatusCondition(metav1.Condition{
			Type:    corednsv1alpha1.RuntimeConditionType,
			Status:  metav1.ConditionFalse,
			Reason:  corednsv1alpha1.ReasonRuntimeInternalError,
			Message: err.Error(),
		})
		return Failed(err)
	}

	dnsctx := DNSContext{
		ReconcileRequest: r,
		Delete:           true,
	}
	dnsdataplane, dnsruntime, prob := r.Reconciler.Options.DNSHandler.Manifests(&dnsctx, values)
	if prob != nil {
		return prob
	}
	if len(dnsruntime) > 0 {
		r.Info("deleting nameserver dns runtime resources")
		for _, data := range runtime {
			prob = AggregateProblem(prob, r.DeleteManifest(r.Reconciler.Runtime, data))
		}
	}
	if len(dnsdataplane) > 0 {
		r.Info("deleting nameserver dns dataplane resources")
		for _, data := range dataplane {
			prob = AggregateProblem(prob, r.DeleteManifest(r.Reconciler.DataPlane, data))
		}
	}
	if prob != nil {
		r.Error("error deleting dns mode objects: {{error}}", "error", prob.Message())
		return prob
	}

	r.Info("deleting runtime resources")
	for _, data := range runtime {
		prob = AggregateProblem(prob, r.DeleteManifest(r.Reconciler.Runtime, data))
	}

	r.Info("deleting dataplane resources")
	for _, data := range dataplane {
		prob = AggregateProblem(prob, r.DeleteManifest(r.Reconciler.DataPlane, data))
	}

	if prob == nil {
		r.Info("cleanup deployment mode")
		prob = r.Reconciler.Mode.Cleanup(r, values["dataplane"].(map[string]interface{})["name"].(string))
		if prob != nil {
			r.Error("error cleanup  mode: {{error}}", "error", prob.Message())
		}
	}
	return prob
}
