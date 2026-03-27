package replicate

import (
	"context"
	"fmt"
	"sync"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubecrtutils/cluster"
	"github.com/mandelsoft/kubecrtutils/owner"
	"github.com/mandelsoft/kubedns/internal/controller/common"
	"github.com/spf13/pflag"
	"sigs.k8s.io/controller-runtime/pkg/client"
	mcreconcile "sigs.k8s.io/multicluster-runtime/pkg/reconcile"
)

type Options struct {
	Class           string
	TargetClass     string
	TargetNamespace string

	OwnerHandler owner.Handler
	lock         sync.RWMutex
	index        map[client.ObjectKey]mcreconcile.Request
}

func From(opts flagutils.OptionSetProvider) *Options {
	return flagutils.GetFrom[*Options](opts)
}

var (
	_ flagutils.Options = (*Options)(nil)
)

func New() *Options {
	return &Options{index: map[client.ObjectKey]mcreconcile.Request{}}
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
	if clusters.Get(SOURCE) == nil {
		return fmt.Errorf("%s cluster is required", SOURCE)
	}
	if clusters.Get(TARGET) == nil {
		return fmt.Errorf("% cluster is required", TARGET)
	}

	com, err := flagutils.ValidatedOptions[*common.Options](ctx, opts, v)
	if err != nil {
		return err
	}
	o.Class = com.Class

	o.OwnerHandler = owner.NewHandler(clusters.Get(TARGET))
	return nil
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&o.TargetNamespace, "target-namespace", "", "", "namespace used to request nameserver DNS names")
	fs.StringVarP(&o.TargetClass, "target-class", "", "", "target class for replication")
}

func (o *Options) GetOriginal(key client.ObjectKey) *mcreconcile.Request {
	o.lock.RLock()
	defer o.lock.RUnlock()
	r, ok := o.index[key]
	if !ok {
		return nil
	}
	return &r
}

func (o *Options) SetOriginal(key client.ObjectKey, tgt mcreconcile.Request) {
	o.lock.Lock()
	defer o.lock.Unlock()
	o.index[key] = tgt
}

func (o *Options) DeleteOriginal(key client.ObjectKey) {
	o.lock.Lock()
	defer o.lock.Unlock()
	delete(o.index, key)
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
