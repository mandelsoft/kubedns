package hostedzone

import (
	"context"
	"fmt"
	"strings"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/goutils/errors"
	"github.com/mandelsoft/kubecrtutils/cluster"
	"github.com/mandelsoft/kubecrtutils/options/manageropts"
	"github.com/mandelsoft/kubedns/internal/controller/common"
	"github.com/spf13/pflag"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type Options struct {
	Runtime          *string
	Class            *string
	DNSClass         string
	DNSDomain        string
	DNSNamespace     string
	DNSMode          string
	RuntimeNamespace string
	Platform         string
	ServerMode       string
	RestEndpoint     string
	Kubedyndns       string
	Restdyndns       string

	DNSHandler DNSHandler
}

func From(opts flagutils.OptionSetProvider) *Options {
	return flagutils.GetFrom[*Options](opts)
}

var (
	_ flagutils.Options                 = (*Options)(nil)
	_ flagutils.Validatable             = (*Options)(nil)
	_ manageropts.ConfigurationProvider = (*Options)(nil)
)

func NewOptions() *Options {
	return &Options{}
}

func (*Options) Prepare(ctx context.Context, opts flagutils.OptionSet, v flagutils.PreparationSet) error {
	return common.Assure(opts)
}

func (o *Options) Validate(ctx context.Context, opts flagutils.OptionSet, v flagutils.ValidationSet) error {
	var err error

	copt := common.From(opts)
	if copt == nil {
		return fmt.Errorf("class option not found in option definitions")
	}
	o.Class = copt.Class
	o.Runtime = copt.Runtime
	clusters, err := cluster.ValidatedClusters(ctx, opts, v)
	if err != nil {
		return err
	}
	if clusters.Get("dataplane") == nil {
		return fmt.Errorf("dataplane cluster is required")
	}
	if clusters.Get("runtime") == nil {
		return fmt.Errorf("dataplane cluster is required")
	}

	o.DNSHandler, err = DNSModes.Create(ctx, o.DNSMode, o)
	return errors.Wrapf(err, "dns mode %q", o.DNSMode)
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	modes := DNSModes.Names()
	fs.StringVarP(&o.RuntimeNamespace, "runtime-namespace", "", "", "use single runtime namespace for deployments")

	fs.StringVarP(&o.DNSMode, "dns-mode", "", "loadbalancer", fmt.Sprintf("DNS mode for providing nameserver cnames [%s]", strings.Join(modes, ",")))
	fs.StringVarP(&o.DNSDomain, "dns-domain", "", "", "DNS domain for managed nameserver DNS names")
	fs.StringVarP(&o.DNSClass, "dns-class", "", "dns-system", "DNS class for managed nameserver DNS names")
	fs.StringVarP(&o.DNSNamespace, "ns-namespace", "", "dns-system", "namespace used to request nameserver DNS names")
	fs.StringVarP(&o.Platform, "iaas", "", "default", "IaaS layer to use (special support so far for \"aws\"")

	fs.StringVarP(&o.ServerMode, "server-mode", "", SERVERMODE_DATAPLANE, fmt.Sprintf("server mode for deployment (%s or %s)", SERVERMODE_RESTAPI, SERVERMODE_DATAPLANE))
	fs.StringVarP(&o.RestEndpoint, "rest-endpoint", "", "", "endpoint for REST API")
	// default images
	fs.StringVarP(&o.Kubedyndns, "kubednydns", "", "mandelsoft:coredns:latest", "image for dns server using dataplane access")
	fs.StringVarP(&o.Restdyndns, "restdnydns", "", "mandelsoft:restdnyndns-coredns:latest", "image for dns server using REST API access")
}

func (o *Options) Configure(ctx context.Context, cfg *manager.Options, opts flagutils.OptionSet) error {
	if o.Runtime != nil && *o.Runtime != "" {
		cfg.LeaderElectionID = *o.Runtime + "-" + cfg.LeaderElectionID
	}
	if o.Class != nil {
		if *o.Class != "" {
			cfg.LeaderElectionID = *o.Class + "-" + cfg.LeaderElectionID
		} else {
			cfg.LeaderElectionID = "default-" + cfg.LeaderElectionID
		}
	}
	return nil
}
