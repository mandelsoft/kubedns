package hostedzone

import (
	"context"
	"fmt"
	"maps"
	"net"

	"github.com/mandelsoft/goutils/sliceutils"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/controller/controllerutils/reconcile"
	"github.com/mandelsoft/kubedns/pkg/render"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	DNSModes.Register("kubedns", DNSModeFactory(NewDNSByKubedns))
}

type dnsKubedns struct {
	domain    string
	class     string
	namespace string
}

func NewDNSByKubedns(ctx context.Context, opts *Options) (DNSHandler, error) {

	if opts.DNSDomain == "" {
		return nil, fmt.Errorf("DNS domain required")
	}
	if opts.DNSClass == "" {
		return nil, fmt.Errorf("DNS class required")
	}
	if opts.DNSNamespace == "" {
		return nil, fmt.Errorf("DNS namespace required")
	}
	return &dnsKubedns{domain: opts.DNSDomain, class: opts.DNSClass, namespace: opts.DNSNamespace}, nil
}

func (d *dnsKubedns) Manifests(ctx *DNSContext, values map[string]interface{}) (map[string][]byte, map[string][]byte, reconcile.Problem) {
	var ips []net.IP
	var cnames []string

	if !ctx.Delete {
		ips, cnames = isLoadBalancerReady(ctx.Service)
		if len(cnames) == 0 && len(ips) == 0 {
			// service change trigger reconcilation -> no backoff
			return nil, nil, reconcile.WatchBackedProblemf("load balancer not yet available")
		}
	}

	dnsnames, err := d.getCNames(ctx)
	if err != nil {
		if !ctx.Delete {
			return nil, nil, reconcile.Requeuef("dnsnames not yet available")
		}
	}

	manifests, err := GetKubeDNSManifests()
	if err != nil {
		if !ctx.Delete {
			return nil, nil, reconcile.Failedf("cannot get dns manifests: %s", err.Error())
		}
	}

	values = maps.Clone(values)
	dns := map[string]interface{}{
		"name":      d.getEntryName(ctx),
		"namespace": d.namespace,
		"dnsnames":  sliceutils.Convert[any](dnsnames),
	}
	if d.class != "" {
		dns["class"] = d.class
	}
	if len(ips) > 0 {
		dns["ips"] = sliceutils.Transform(ips, func(ip net.IP) any { return net.IP.String(ip) })
	}
	if len(cnames) > 0 {
		dns["cname"] = cnames[0]
	}
	values["dns"] = dns
	dataplane, runtime, err := render.Render(manifests, values)
	if err != nil {
		return nil, nil, reconcile.Failedf("cannot get dns manifests: %s", err.Error())
	}

	return dataplane, runtime, nil
}

func (d *dnsKubedns) Modify(ctx *DNSContext, obj client.Object) error {
	return nil
}

func (d *dnsKubedns) GetCNames(ctx *DNSContext) ([]string, reconcile.Problem) {
	ips, cnames := isLoadBalancerReady(ctx.Service)
	if len(cnames) == 0 && len(ips) == 0 {
		// service change trigger reconcilation -> no backoff
		return nil, reconcile.WatchBackedProblemf("load balancer not yet available")
	}

	var entry corednsv1alpha1.CoreDNSEntry
	err := ctx.reconciler.DataPlane.Get(ctx, client.ObjectKey{Namespace: "dns-system", Name: d.getEntryName(ctx)}, &entry)
	if err != nil {
		return nil, reconcile.TemporaryProblem(err)
	}
	if entry.Status.State != corednsv1alpha1.STATE_READY && entry.Status.State != corednsv1alpha1.STATE_OK {
		return nil, reconcile.TemporaryProblemf("dns entry for nameserver not yet ready: %s", entry.Status.Message)
	}
	cnames, err = d.getCNames(ctx)
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

func (d *dnsKubedns) getCNames(ctx *DNSContext) ([]string, error) {
	n := fmt.Sprintf("%s.%s.%s", ctx.Name, ctx.Namespace, d.domain)
	return []string{n}, nil
}

func (d *dnsKubedns) getEntryName(ctx *DNSContext) string {
	return ctx.Namespace + "-" + ctx.Name
}
