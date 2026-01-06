package owner

import (
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/types"
	"k8s.io/apimachinery/pkg/runtime"
)

type RemoteFactory func(scheme *runtime.Scheme, cluster string) OwnerHandler

func For(owner types.Cluster, slave types.Cluster, fac ...RemoteFactory) OwnerHandler {
	return ForNames(owner.GetScheme(), owner.GetId(), slave.GetEffective().GetName(), fac...)
}

func ForNames(scheme *runtime.Scheme, owner string, slave string, fac ...RemoteFactory) OwnerHandler {
	remote := CreateRemoteHandler(scheme, owner, fac...)
	if owner == slave {
		return LocalOwner(scheme, remote)
	} else {
		return remote
	}
}
