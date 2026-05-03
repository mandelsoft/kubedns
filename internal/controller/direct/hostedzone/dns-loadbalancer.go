package hostedzone

import (
	"context"

	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconcile"
)

const DNSMODE_LOADBALANCER = "loadbalancer"

func init() {
	DNSModes.Register(DNSMODE_LOADBALANCER, NewLoadbalancerDNSModeFactory())
}

type loadbalancerDNSModeFactory struct {
}

func NewLoadbalancerDNSModeFactory() DNSModeFactory {
	return &loadbalancerDNSModeFactory{}
}

func (s *loadbalancerDNSModeFactory) Description() string {
	return "Loadbalancer DNS names"
}

func (s *loadbalancerDNSModeFactory) Create(ctx context.Context, cfg *Options) (DNSMode, error) {
	return &dnsLoadbalancer{DNSDummy{DNSMODE_LOADBALANCER}}, nil
}

func (f *loadbalancerDNSModeFactory) IsDefault() bool {
	return true
}

////////////////////////////////////////////////////////////////////////////////

type dnsLoadbalancer struct {
	DNSDummy
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
