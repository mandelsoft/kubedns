package hostedzone

import (
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/controller/controllerutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/controller/controllerutils/reconcile"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DNSModeFactory = controllerutils.FactoryFunc[*Options, DNSHandler]

var DNSModes = controllerutils.NewRegistry[*Options, DNSHandler]("DNS mode")

type DNSContext struct {
	*ReconcileRequest
	Service *v1.Service
}

type DNSHandler interface {
	Manifests(ctx *DNSContext, values map[string]interface{}) [][]byte
	Modify(ctx *DNSContext, obj client.Object) error
	GetCNames(ctx *DNSContext) (cnames []string, prob reconcile.Problem)
}
