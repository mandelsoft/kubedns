package hostedzone

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/mandelsoft/kubedns/pkg/objutils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	DNSModes.Register("gardener", DNSModeFactory(NewDNSByGardener))
}

type dnsGardener struct {
	domain string
	class  string
}

func NewDNSByGardener(ctx context.Context, opts *Options) (DNSHandler, error) {
	class := opts.DNSClass
	if class == "" {
		class = "garden"
	}
	if opts.DNSDomain == "" {
		return nil, fmt.Errorf("DNS domain required")
	}
	return &dnsGardener{domain: opts.DNSDomain, class: class}, nil
}

func (d *dnsGardener) Manifests(ctx *DNSContext, values map[string]interface{}) [][]byte {
	return nil
}

func (d *dnsGardener) Modify(ctx *DNSContext, obj client.Object) error {
	gvk := obj.GetObjectKind().GroupVersionKind()
	if gvk.Kind != "Service" {
		return nil
	}
	if gvk.Group != "" {
		return nil
	}
	cnames, err := d.getCNames(ctx)
	if err != nil {
		return err
	}

	ctx.Info("adding gardener annotations for {{domain}} and class {{class}}", "domain", cnames, "class", d.class)
	objutils.SetAnnotation(obj, "dns.gardener.cloud/dnsnames", strings.Join(cnames, ","))
	objutils.SetAnnotation(obj, "dns.gardener.cloud/class", d.class)
	return nil
}

func (d *dnsGardener) GetCNames(ctx *DNSContext) ([]string, error, error) {
	ips, cnames := isLoadBalancerReady(ctx.Service)
	if len(cnames) == 0 && len(ips) == 0 {
		// service change trigger reconcilation -> no backoff
		return nil, fmt.Errorf("load balancer not yet available"), nil
	}
	cnames, err := d.getCNames(ctx)
	if err != nil {
		return nil, nil, err
	}
	for _, n := range cnames {
		if isResolvable(n) {
			return cnames, nil, nil
		}
	}
	return nil, nil, fmt.Errorf("cnames not reachable, so far")
}

func (d *dnsGardener) getCNames(ctx *DNSContext) ([]string, error) {
	n := fmt.Sprintf("%s.%s.%s", ctx.Name, ctx.Namespace, d.domain)
	return []string{n}, nil
}

func isResolvable(domain string) bool {
	// LookupHost returns the host's addresses
	ips, err := net.LookupHost(domain)
	if err != nil {
		return false
	}
	return len(ips) > 0
}
