package controller

import (
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cacheindex"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/types"
)

type Controllers = types.Controllers

type Controller[T any, P kubecrtutils.ObjectPointer[T]] interface {
	types.Controller
	GetDefinition() TypedDefinition[T, P]
	GetTypedIndex(name string) cacheindex.Index[T]
}
