package down

import (
	"github.com/mandelsoft/kubecrtutils/controller"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/replicate"
	"github.com/mandelsoft/kubedns/internal/controller/replicate/entry/common/down"
)

func Controller() controller.Definition {
	return down.Controller[*corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry](replicate.ControllerEntry, replicate.ENTRY_GROUP)
}
