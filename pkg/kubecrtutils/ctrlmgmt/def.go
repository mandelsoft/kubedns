package ctrlmgmt

import (
	"errors"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cluster"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/controller"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/index"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/internal"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/options/manageropts"
	"github.com/spf13/pflag"
	"golang.org/x/net/context"
)

func From(opts flagutils.OptionSetProvider) Definition {
	return flagutils.GetFrom[Definition](opts)
}

type Definition interface {
	internal.Named
	flagutils.Options
	flagutils.OptionSetProvider
	AddCluster(def ...cluster.Definition) Definition
	AddController(def ...controller.Definition) Definition
	AddIndex(def ...index.Definition) Definition

	GetError() error
}

type definition struct {
	internal.Element
	options     flagutils.DefaultOptionSet
	clusters    cluster.Definitions
	controllers controller.Definitions
	indices     index.Definitions
}

func NewDefinition(name, main string) Definition {
	d := &definition{
		Element:     internal.NewElement(name),
		clusters:    cluster.NewDefinitions(),
		indices:     index.NewDefinitions(),
		controllers: controller.NewDefinitions(),
	}
	d.options.Add(d.clusters, d.indices, manageropts.New(main, name))
	return d
}

func (d *definition) AddCluster(def ...cluster.Definition) Definition {
	d.clusters.Add(def...)
	return d
}

func (d *definition) AddController(def ...controller.Definition) Definition {
	d.controllers.Add(def...)
	return d
}

func (d *definition) AddIndex(def ...index.Definition) Definition {
	d.indices.Add(def...)
	return d
}

func (d *definition) AddFlags(fs *pflag.FlagSet) {
	for o := range d.options.Options {
		o.AddFlags(fs)
	}
}

func (d *definition) AsOptionSet() flagutils.OptionSet {
	return d.options
}

func (d *definition) GetError() error {
	return errors.Join(d.clusters.GetError(), d.indices.GetError(), d.controllers.GetError())
}

func (d *definition) GetControllerManager(ctx context.Context, opts flagutils.OptionSetProvider) (ControllerManager, error) {
	return NewControllerManagerByOpts(ctx, opts.AsOptionSet())
}
