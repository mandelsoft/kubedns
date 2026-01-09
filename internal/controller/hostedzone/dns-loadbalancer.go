package hostedzone

import (
	"context"

	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/controller/controllerutils/reconcile"
)

func init() {
	DNSModes.Register("loadbalancer", DNSModeFactory(NewDNSByLoadBalancer))
}

type dnsLoadbalancer struct {
	DNSDummy
}

func NewDNSByLoadBalancer(ctx context.Context, opts *Options) (DNSHandler, error) {
	return &dnsLoadbalancer{}, nil
}

func (l *dnsLoadbalancer) GetCNames(ctx *DNSContext) ([]string, reconcile.Problem) {
	ips, cnames := isLoadBalancerReady(ctx.Service)
	if len(cnames) > 0 {
		return cnames, nil
	}
	if len(ips) > 0 {
		return nil, reconcile.WatchBackedProblemf("no cnames available for load balancer")
	}
	return nil, reconcile.WatchBackedProblemf("load balancer not yet available")
}
