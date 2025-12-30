package common

import (
	"github.com/mandelsoft/kubedns/pkg/clusterutils"
	ctrl "sigs.k8s.io/controller-runtime"
)

type ReconcilerContext struct {
	Manager  ctrl.Manager
	Clusters clusterutils.Clusters
}
