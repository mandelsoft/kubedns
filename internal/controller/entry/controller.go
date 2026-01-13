package entry

import (
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/common"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/controller"
)

func Controller() controller.Definition {
	return controller.DefineByFunc[corednsv1alpha1.CoreDNSEntry](common.ControllerEntry, "dataplane", CreateReconciler).
		UseCluster("runtime").
		AddIndex(common.IndexKeyEntryZone, zoneIndexer)
}

func zoneIndexer(res *corednsv1alpha1.CoreDNSEntry) []string {
	if res.Spec.ZoneRef == "" {
		return nil
	}
	return []string{res.Spec.ZoneRef}
}
