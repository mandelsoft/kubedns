package down

import (
	"github.com/mandelsoft/kubecrtutils/controller"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/replicate"
	"github.com/mandelsoft/kubedns/internal/controller/replicate/common"
	"github.com/mandelsoft/kubedns/internal/controller/replicate/common/down"
)

func Controller() controller.Definition {
	return down.Controller[*corednsv1alpha1.HostedZone, corednsv1alpha1.HostedZone](
		replicate.ControllerHostedzone,
		replicate.HOSTEDZONE_GROUP,
		common.ProviderFunc(GetMapping),
	)
}

func GetMapping(o *common.Options) common.Mapping {
	return o.Zones
}
