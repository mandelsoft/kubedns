package hostedzone

import (
	"context"
	"fmt"
	"strings"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cluster"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/options/manageropts"
	"github.com/spf13/pflag"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Options struct {
	Runtime          string
	Class            string
	DNSClass         string
	DNSDomain        string
	DNSNamespace     string
	DNSMode          string
	RuntimeNamespace string
	Platform         string

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

func (o *Options) Validate(ctx context.Context, opts flagutils.OptionSet, v flagutils.ValidationSet) error {
	var err error

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
	return err
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	modes := DNSModes.Names()
	fs.StringVarP(&o.RuntimeNamespace, "runtime-namespace", "", "", "use single runtime namespace for deployments")
	fs.StringVarP(&o.Runtime, "runtime", "", "", "name of the runtime class to handle")
	fs.StringVarP(&o.Class, "class", "", "", "name of the controller class to handle")

	fs.StringVarP(&o.DNSMode, "dns-mode", "", "loadbalancer", fmt.Sprintf("DNS mode for providing nameserver cnames [%s]", strings.Join(modes, ",")))
	fs.StringVarP(&o.DNSDomain, "dns-domain", "", "", "DNS domain for managed nameserver DNS names")
	fs.StringVarP(&o.DNSClass, "dns-class", "", "", "DNS class for managed nameserver DNS names")
	fs.StringVarP(&o.DNSNamespace, "ns-namespace", "", "", "namespace used to request nameserver DNS names")
	fs.StringVarP(&o.Platform, "iaas", "", "default", "IaaS layer to use (special support so far for \"aws\"")
}

func (o *Options) Configure(ctx context.Context, cfg *ctrl.Options, opts flagutils.OptionSet, v flagutils.ValidationSet) error {
	if o.Runtime != "" {
		cfg.LeaderElectionID = o.Runtime + "-" + cfg.LeaderElectionID
	}
	if o.Class != "" {
		cfg.LeaderElectionID = o.Class + "-" + cfg.LeaderElectionID
	}
	return nil
}
