package hostedzone

import (
	"context"
	"errors"
	"fmt"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubecrtutils/cluster"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils"
	"github.com/mandelsoft/kubecrtutils/options/manageropts"
	"github.com/mandelsoft/kubedns/internal/controller/common"
	"github.com/spf13/pflag"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type Options struct {
	Runtime          *string
	Class            *string
	RuntimeNamespace string
	Platform         string

	_DNSModes    controllerutils.Registry[*Options, DNSMode]
	_ServerModes controllerutils.Registry[*Options, ServerMode]

	DNSMode    DNSMode
	ServerMode ServerMode
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
	return &Options{
		_DNSModes:    DNSModes.Clone(),
		_ServerModes: ServerModes.Clone(),
	}
}

func (o *Options) Prepare(ctx context.Context, opts flagutils.OptionSet, v flagutils.PreparationSet) error {
	return errors.Join(
		v.PrepareSet(ctx, opts, o._DNSModes),
		v.PrepareSet(ctx, opts, o._ServerModes),
		common.Assure(opts),
	)
}

func (o *Options) Validate(ctx context.Context, opts flagutils.OptionSet, v flagutils.ValidationSet) error {
	err := errors.Join(
		v.ValidateSet(ctx, opts, o._DNSModes),
		v.ValidateSet(ctx, opts, o._ServerModes),
	)
	if err != nil {
		return err
	}
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

	o.DNSMode, err = o._DNSModes.CreateConfigured(ctx, o)
	if err != nil {
		return err
	}
	o.ServerMode, err = o._ServerModes.CreateConfigured(ctx, o)
	return err
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	o._ServerModes.AddFlags(fs)
	o._DNSModes.AddFlags(fs)
	fs.StringVarP(&o.RuntimeNamespace, "runtime-namespace", "", "", "use single runtime namespace for deployments")
	fs.StringVarP(&o.Platform, "iaas", "", "default", "IaaS layer to use (special support so far for \"aws\"")
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
