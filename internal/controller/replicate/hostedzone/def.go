package hostedzone

import (
	"github.com/mandelsoft/kubecrtutils/cacheindex"
	"github.com/mandelsoft/kubecrtutils/controller"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/common"
	"github.com/mandelsoft/kubedns/internal/controller/replicate"
	"github.com/mandelsoft/kubedns/internal/controller/replicate/common/generic"
)

func Controller() controller.Definition {
	return generic.Controller[*corednsv1alpha1.HostedZone, corednsv1alpha1.HostedZone](
		replicate.ControllerHostedzone,
		replicate.GROUP,
		Responsibility,
	).
		AddIndex(replicate.IndexKeyZoneParent, common.ParentIndexer).
		AddForeignIndex(cacheindex.Define[*corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry](replicate.IndexKeyEntryZone, replicate.SOURCE, common.ZoneIndexer))
}
