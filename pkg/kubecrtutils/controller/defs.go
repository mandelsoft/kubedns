package controller

import (
	"context"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/internal"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/types"
)

func From(opts flagutils.OptionSetProvider) Definitions {
	return flagutils.GetFrom[Definitions](opts)
}

type Definitions interface {
	internal.Definitions[Definition, Definitions]

	Apply(ctx context.Context, manager types.ControllerManager) (Controllers, error)
}

type _definitions struct {
	internal.DefinitionsImpl[Definition, Definitions]
}

func NewDefinitions() Definitions {
	d := &_definitions{}
	d.DefinitionsImpl = internal.NewDefinitions[Definition, Definitions]("index", d)
	return d
}

func (d *_definitions) Apply(ctx context.Context, mgr types.ControllerManager) (Controllers, error) {
	controllers := NewControllers()
	for _, i := range d.Elements {
		c, err := i.Apply(ctx, mgr)
		if err != nil {
			return nil, err
		}
		err = controllers.Add(c)
		if err != nil {
			return nil, err
		}
	}
	return controllers, nil
}
