package cluster

import (
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/internal"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/types"
)

////////////////////////////////////////////////////////////////////////////////

type clusters struct {
	internal.Group[types.Cluster]
}

var _ Clusters = (*clusters)(nil)

func NewClusters() Clusters {
	return &clusters{internal.NewGroup[types.Cluster]("cluster")}
}
