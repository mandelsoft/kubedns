package hostedzone

import (
	"context"
	"fmt"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/goutils/general"
	"github.com/mandelsoft/kubedns/pkg/options/kubeconfigopts"
	"github.com/mandelsoft/kubedns/pkg/options/manageropts"
	"github.com/spf13/pflag"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Options struct {
	Runtime          string
	Class            string
	DNSDomain        string
	DNSNamespace     string
	DNSMode          string
	RuntimeNamespace string
	Platform         string

	RuntimeConfig *kubeconfigopts.Options

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

func NewOptions(def ...*kubeconfigopts.Options) *Options {
	return &Options{
		RuntimeConfig: kubeconfigopts.New("use separated runtime cluster", "runtime").WithFallback(general.Optional(def...)),
	}
}

func (o *Options) Validate(ctx context.Context, opts flagutils.OptionSet, v flagutils.ValidationSet) error {
	switch o.DNSMode {
	case "loadbalancer":
		o.DNSHandler = NewDNSByLoadBalancer()
	// case "gardener":
	// case "local":
	default:
		return fmt.Errorf("unsupported ns-mode %q", o.DNSMode)
	}
	return o.RuntimeConfig.Validate(ctx, opts, v)
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	o.RuntimeConfig.AddFlags(fs)
	fs.StringVarP(&o.RuntimeNamespace, "runtime-namespace", "", "", "use single runtime namespace for deployments")
	fs.StringVarP(&o.Runtime, "runtime", "", "", "name of the runtime class to handle")
	fs.StringVarP(&o.Class, "class", "", "", "name of the controller class to handle")

	fs.StringVarP(&o.DNSMode, "ns-mode", "", "loadbalancer", "DNS mode for providing nameserver cnames")
	fs.StringVarP(&o.DNSDomain, "ns-domain", "", "", "DNS domain for managed nameserver DNS names")
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

func (o *Options) GetRestConfig() *rest.Config {
	return o.RuntimeConfig.GetRestConfig()
}
