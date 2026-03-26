package entry

import (
	"github.com/mandelsoft/kubecrtutils/cacheindex"
	"github.com/mandelsoft/kubecrtutils/controller"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/common"
)

func Controller() controller.Definition {
	return controller.DefineByFunc[*corednsv1alpha1.CoreDNSEntry](common.ControllerEntry, "dataplane", CreateReconciler).
		UseCluster("runtime").
		InGroup("functional").
		AddIndex(common.IndexKeyEntryZone, zoneIndexer).
		ImportIndex(cacheindex.Ref[*corednsv1alpha1.HostedZone, corednsv1alpha1.HostedZone](common.IndexKeyZoneParent, "dataplane"))
}

func zoneIndexer(res *corednsv1alpha1.CoreDNSEntry) []string {
	if res.Spec.ZoneRef == "" {
		return nil
	}
	return []string{res.Spec.ZoneRef}
}
