package common

import (
	"context"
	"fmt"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubecrtutils/cluster"
	"github.com/mandelsoft/kubecrtutils/controller"
	"github.com/mandelsoft/kubecrtutils/owner"
	"github.com/mandelsoft/kubedns/internal/controller/common"
	"github.com/mandelsoft/kubedns/internal/controller/replicate"
	"github.com/spf13/pflag"
)

type Options struct {
	Class           string
	Runtime         *string
	TargetClass     string
	TargetNamespace string

	OwnerHandler owner.Handler
	*ReplicationMapping
}

func From(opts flagutils.OptionSetProvider) *Options {
	return flagutils.GetFrom[*Options](opts)
}

var (
	_ flagutils.Options            = (*Options)(nil)
	_ controller.FinalizerModifier = (*Options)(nil)
)

func New() *Options {
	return &Options{ReplicationMapping: NewReplicationMapping()}
}

func (o *Options) ModifyFinalizer(f string) string {
	if o.Class != "" {
		f = f + "." + o.Class
	}
	if o.Runtime != nil && *o.Runtime != "" {
		f = f + "." + *o.Runtime
	}
	return f
}

func (o *Options) Prepare(ctx context.Context, opts flagutils.OptionSet, v flagutils.PreparationSet) error {
	return common.Assure(opts)
}

func (o *Options) Validate(ctx context.Context, opts flagutils.OptionSet, v flagutils.ValidationSet) error {
	var err error

	clusters, err := cluster.ValidatedClusters(ctx, opts, v)
	if err != nil {
		return err
	}
	if clusters.Get(replicate.SOURCE) == nil {
		return fmt.Errorf("%s cluster is required", replicate.SOURCE)
	}
	if clusters.Get(replicate.TARGET) == nil {
		return fmt.Errorf("% cluster is required", replicate.TARGET)
	}

	com, err := flagutils.ValidatedOptions[*common.Options](ctx, opts, v)
	if err != nil {
		return err
	}
	o.Class = com.Class
	o.Runtime = com.Runtime

	o.OwnerHandler = owner.NewHandler(clusters.Get(replicate.TARGET))
	return nil
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&o.TargetNamespace, "target-namespace", "", "", "namespace used to request nameserver DNS names")
	fs.StringVarP(&o.TargetClass, "target-class", "", "", "target class for replication")
}

////////////////////////////////////////////////////////////////////////////////

type OptionsRef = flagutils.OptionsRef[*Options]

func NewOptionsRef() *OptionsRef {
	return flagutils.NewOptionsRef[*Options](New)
}

///////////////////////////////////////////////////////////////////////////////

type Factory struct {
}

func (f Factory) CreateOptions() *Options {
	return New()
}
