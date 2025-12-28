package hostedzone

import (
	"context"

	"github.com/mandelsoft/kubedns/pkg/controllerutils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	DNSModes.Register("loadbalancer", DNSModeFactory(NewDNSByLoadBalancer))
}

type dnsLoadbalancer struct {
}

func NewDNSByLoadBalancer(ctx context.Context, opts *Options) (DNSHandler, error) {
	return &dnsLoadbalancer{}, nil
}

func (l *dnsLoadbalancer) Manifests(ctx *DNSContext, values map[string]interface{}) [][]byte {
	return nil
}

func (l *dnsLoadbalancer) Modify(ctx *DNSContext, obj client.Object) error {
	return nil
}

func (l *dnsLoadbalancer) GetCNames(ctx *DNSContext) ([]string, error) {
	ips, cnames := isLoadBalancerReady(ctx.Service)
	if len(cnames) > 0 {
		return cnames, nil
	}
	if len(ips) > 0 {
		return nil, controllerutils.RequestRequeuef("no cnames available for load balancer")
	}
	return nil, nil
}
