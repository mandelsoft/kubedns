package common

import (
	"github.com/mandelsoft/kubecrtutils/cluster"
	ctrl "sigs.k8s.io/controller-runtime"
)

type ReconcilerContext struct {
	Manager  ctrl.Manager
	Clusters cluster.Clusters
}
