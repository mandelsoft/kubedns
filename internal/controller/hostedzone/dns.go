package hostedzone

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DNSContext struct {
	*ReconcileRequest
	Service *v1.Service
}
type DNSHandler interface {
	Manifests(ctx *DNSContext, values map[string]interface{}) [][]byte
	Modify(ctx *DNSContext, obj client.Object) error
	GetCNames(ctx *DNSContext) ([]string, error)
}

////////////////////////////////////////////////////////////////////////////////

type loadbalancer struct {
}

func NewDNSByLoadBalancer() DNSHandler {
	return &loadbalancer{}
}

func (l *loadbalancer) Manifests(ctx *DNSContext, values map[string]interface{}) [][]byte {
	return nil
}

func (l *loadbalancer) Modify(ctx *DNSContext, obj client.Object) error {
	return nil
}

func (l *loadbalancer) GetCNames(ctx *DNSContext) ([]string, error) {
	ips, cnames := isLoadBalancerReady(ctx.Service)
	if len(cnames) > 0 {
		return cnames, nil
	}
	if len(ips) > 0 {
		return nil, fmt.Errorf("no cnames available for load balancer")
	}
	return nil, nil
}
