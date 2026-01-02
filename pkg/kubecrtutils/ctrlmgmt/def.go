package ctrlmgmt

import (
	"errors"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cluster"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/index"
	"github.com/spf13/pflag"
)

type Definition interface {
	flagutils.Options
	flagutils.OptionSetProvider
	AddCluster(def ...cluster.Definition) Definition
}

type definition struct {
	options  flagutils.DefaultOptionSet
	clusters cluster.Definitions
	indices  index.Definitions
}

func NewDefinition() Definition {
	d := &definition{
		clusters: cluster.NewDefinitions(),
		indices:  index.NewDefinitions(),
	}
	d.options.Add(d.clusters, d.indices)
	return d
}

func (d *definition) GetError() error {
	return errors.Join(d.clusters.GetError(), d.indices.GetError())
}

func (d *definition) AddCluster(def ...cluster.Definition) Definition {
	d.clusters.Add(def...)
	return d
}

func (d *definition) AddIndex(def ...index.Definition) Definition {
	d.indices.Add(def...)
	return d
}

func (d *definition) AddFlags(fs *pflag.FlagSet) {
	d.clusters.AddFlags(fs)
	d.indices.AddFlags(fs)
}

func (d *definition) AsOptionSet() flagutils.OptionSet {
	return d.options
}
