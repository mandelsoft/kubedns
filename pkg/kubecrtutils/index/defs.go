package index

import (
	"context"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cluster"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/internal"
)

func From(opts flagutils.OptionSetProvider) Definitions {
	return flagutils.GetFrom[Definitions](opts)
}

type Definitions interface {
	internal.Definitions[Definition, Definitions]

	GetIndices(ctx context.Context, clusters cluster.Clusters) (Indices, error)
}

type _definitions struct {
	internal.DefinitionsImpl[Definition, Definitions]
}

func NewDefinitions() Definitions {
	d := &_definitions{}
	d.DefinitionsImpl = internal.NewDefinitions[Definition, Definitions]("index", d)
	return d
}

func (d *_definitions) GetIndices(ctx context.Context, clusters cluster.Clusters) (Indices, error) {
	indices := NewIndices()
	for _, i := range d.Elements {
		_, err := i.Apply(ctx, clusters)
		if err != nil {
			return nil, err
		}
	}
	return indices, nil
}
