package entry

import (
	"github.com/mandelsoft/kubecrtutils/cacheindex"
	"github.com/mandelsoft/kubecrtutils/controller"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/direct"
)

func Controller() controller.Definition {
	return controller.DefineByFunc[*corednsv1alpha1.CoreDNSEntry](direct.ControllerEntry, "dataplane", CreateReconciler).
		UseCluster("runtime").
		InGroup(direct.GROUP).
		AddIndex(direct.IndexKeyEntryZone, direct.ZoneIndexer).
		ImportIndex(cacheindex.Ref[*corednsv1alpha1.HostedZone, corednsv1alpha1.HostedZone](direct.IndexKeyZoneParent, "dataplane"))
}

func zoneIndexer(res *corednsv1alpha1.CoreDNSEntry) []string {
	if res.Spec.ZoneRef == "" {
		return nil
	}
	return []string{res.Spec.ZoneRef}
}
