package controller

import (
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/internal"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/types"
)

type _controllers struct {
	internal.Group[types.Controller]
}

func NewControllers() types.Controllers {
	return &_controllers{internal.NewGroup[types.Controller]("controller")}
}
