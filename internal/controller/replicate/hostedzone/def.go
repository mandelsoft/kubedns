package hostedzone

import (
	"github.com/mandelsoft/kubecrtutils/cacheindex"
	"github.com/mandelsoft/kubecrtutils/controller"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/replicate"
	"github.com/mandelsoft/kubedns/internal/controller/replicate/common/generic"
)

func Controller() controller.Definition {
	return generic.Controller[*corednsv1alpha1.HostedZone, corednsv1alpha1.HostedZone](
		replicate.ControllerHostedzone,
		replicate.HOSTEDZONE_GROUP,
		Responsibility,
	).
		AddIndex(replicate.IndexKeyZoneParent, parentIndexer).
		AddForeignIndex(cacheindex.Define[*corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry](replicate.IndexKeyEntryZone, replicate.SOURCE, zoneIndexer))
}

func zoneIndexer(res *corednsv1alpha1.CoreDNSEntry) []string {
	if res.Spec.ZoneRef == "" {
		return nil
	}
	return []string{res.Spec.ZoneRef}
}

func parentIndexer(o *corednsv1alpha1.HostedZone) []string {
	if o.Spec.ParentRef == "" {
		return nil
	}
	return []string{o.Spec.ParentRef}
}
