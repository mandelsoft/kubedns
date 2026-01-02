package index

import (
	"context"

	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cluster"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/internal"
)

type Definitions interface {
	internal.Definitions[Definition, Definitions]

	Apply(ctx context.Context, clusters cluster.Clusters) error
}

type _definitions struct {
	internal.DefinitionsImpl[Definition, Definitions]
}

func NewDefinitions() Definitions {
	d := &_definitions{}
	d.DefinitionsImpl = internal.NewDefinitions[Definition, Definitions]("index", d)
	return d
}

func (d *_definitions) Apply(ctx context.Context, clusters cluster.Clusters) error {
	for _, i := range d.Elements {
		_, err := i.Apply(ctx, clusters)
		if err != nil {
			return err
		}
	}
	return nil
}
