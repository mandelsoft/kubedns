package hostedzone

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconcile"
	"github.com/mandelsoft/kubecrtutils/objutils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const DNSMODE_GARDENER = "gardener"

func init() {
	DNSModes.Register(DNSMODE_GARDENER, NewGardenerDNSModeFactory())
}

var (
	_ flagutils.OptionSet = (*gardenerDNSModeFactory)(nil)
)

type gardenerDNSModeFactory struct {
	flagutils.DefaultOptionSet
	class  *flagutils.OptionsRef[*DNSClassOption]
	domain *flagutils.OptionsRef[*DNSDomainOption]
}

func NewGardenerDNSModeFactory() DNSModeFactory {
	f := &gardenerDNSModeFactory{
		class:  flagutils.NewDefaultOptionsRef[*DNSClassOption](),
		domain: flagutils.NewDefaultOptionsRef[*DNSDomainOption](),
	}
	f.Add(f.class, f.domain)
	return f
}

func (s *gardenerDNSModeFactory) Description() string {
	return "Gardener DNS record provisioning"
}

func (s *gardenerDNSModeFactory) Create(ctx context.Context, cfg *Options) (DNSMode, error) {
	class := s.class.Options.class
	if class == "" {
		class = "garden"
	}
	if s.domain.Options.domain == "" {
		return nil, fmt.Errorf("DNS domain required")
	}
	return &dnsGardener{DNSDummy: DNSDummy{DNSMODE_GARDENER}, domain: s.domain.Options.domain, class: class}, nil
}

////////////////////////////////////////////////////////////////////////////////

type dnsGardener struct {
	DNSDummy
	domain string
	class  string
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
	objutils.SetAnnotation(obj, "dns.gardener.cloud/ttl", "60")
	return nil
}

func (d *dnsGardener) GetCNames(ctx *DNSContext) ([]string, reconcile.Problem) {
	ips, cnames := isLoadBalancerReady(ctx.Service)
	if len(cnames) == 0 && len(ips) == 0 {
		// service change trigger reconcilation -> no backoff
		return nil, reconcile.WatchBackedProblemf("load balancer not yet available")
	}
	cnames, err := d.getCNames(ctx)
	if err != nil {
		return nil, reconcile.WatchBackedProblem(err)
	}
	for _, n := range cnames {
		if isResolvable(n) {
			return cnames, nil
		}
	}
	return nil, reconcile.Requeuef("cnames not reachable, so far")
}

func (d *dnsGardener) getCNames(ctx *DNSContext) ([]string, error) {
	key := ctx.GetKey()
	n := fmt.Sprintf("%s.%s.%s", key.Name, key.Namespace, d.domain)
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
