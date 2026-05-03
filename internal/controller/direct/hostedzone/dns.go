package hostedzone

import (
	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconcile"
	"github.com/mandelsoft/kubedns/pkg/render"
	"github.com/spf13/pflag"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DNSModeFactory = controllerutils.Factory[*Options, DNSMode]

var DNSModes = controllerutils.NewRegistry[*Options, DNSMode]("dns-mode", "how to provide domains for DNS servers")

type DNSContext struct {
	*ReconcileRequest
	Service *v1.Service
	Delete  bool
}

type DNSMode interface {
	GetName() string
	Manifests(ctx *DNSContext, values map[string]interface{}) (*render.Rendered, reconcile.Problem)
	Modify(ctx *DNSContext, obj client.Object) error
	GetCNames(ctx *DNSContext) (cnames []string, prob reconcile.Problem)
}

type DNSDummy struct {
	name string
}

func (d DNSDummy) GetName() string {
	return d.name
}

func (DNSDummy) Manifests(ctx *DNSContext, values map[string]interface{}) (*render.Rendered, reconcile.Problem) {
	return nil, nil
}
func (DNSDummy) Modify(ctx *DNSContext, obj client.Object) error {
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Options shared by multiple DNSModes

type DNSClassOption struct {
	class string
}

var _ flagutils.Options = (*DNSClassOption)(nil)

func (o *DNSClassOption) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.class, "dns-class", "", "DNS class for managed nameserver DNS names")
}

////////////////////////////////////////////////////////////////////////////////

type DNSDomainOption struct {
	domain string
}

var _ flagutils.Options = (*DNSDomainOption)(nil)

func (o *DNSDomainOption) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.domain, "dns-domain", "", "DNS domain for managed nameserver DNS names")
}
