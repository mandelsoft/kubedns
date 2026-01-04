package index

import (
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/internal"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/types"
)

type _indices struct {
	internal.Group[types.Index]
}

func NewIndices() Indices {
	return &_indices{internal.NewGroup[types.Index]("index")}
}
