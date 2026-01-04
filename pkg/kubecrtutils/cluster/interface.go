package cluster

import (
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/types"
	"k8s.io/apimachinery/pkg/runtime"
)

type SchemeProvider interface {
	GetScheme() *runtime.Scheme
}

type Cluster = types.Cluster
type Clusters = types.Clusters
type Index = types.Index
