package controller

import (
	"context"
	"fmt"

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
	mgr.GetLogger().Info("configure controllers...")
	// Step 1: create controllers and their environment like indices
	for n, i := range d.Elements {
		c, err := i.CreateController(ctx, mgr)
		if err != nil {
			return nil, fmt.Errorf("controller %q: %w", n, err)
		}
		err = controllers.Add(c)
		if err != nil {
			return nil, fmt.Errorf("controller %q: %w", n, err)
		}
	}
	// Step 2: complete the controller by creating their reconciler
	// (by factory) and finally configure the controller runtime manager.
	for n, i := range controllers.Elements {
		err := i.Complete(ctx)
		if err != nil {
			return nil, fmt.Errorf("controller %q: %w", n, err)
		}
	}
	return controllers, nil
}
