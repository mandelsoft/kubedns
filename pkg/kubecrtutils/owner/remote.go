package owner

import (
	"fmt"
	"strings"

	"github.com/mandelsoft/goutils/general"
	"github.com/mandelsoft/kubedns/pkg/objutils"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

const DEFAULT_REMOTE_NAME = "cross-cluster.io/owner-id"

func CreateRemoteHandler(scheme *runtime.Scheme, cluster string, fac ...RemoteFactory) OwnerHandler {
	f := general.Optional(fac...)
	if f == nil {
		return RemotePropertyOwner(scheme, cluster)
	}
	return f(scheme, cluster)
}

type remote struct {
	scheme   *runtime.Scheme
	property string
	cluster  string
}

func RemotePropertyOwner(scheme *runtime.Scheme, ownerCluster string, property ...string) OwnerHandler {
	return &remote{scheme: scheme, cluster: ownerCluster, property: general.OptionalNonZeroDefaulted(DEFAULT_REMOTE_NAME, ownerCluster)}
}

func (l *remote) GetOwner(obj client.Object, kind schema.GroupKind) *client.ObjectKey {
	if kind.Group == "" {
		kind.Group = "core"
	}

	// group / kind / namespace / name / cluster
	a := objutils.GetAnnotation(obj, l.property)
	if a == "" {
		return nil
	}
	fields := strings.Split(a, "/")
	if l.match(fields, kind) {
		return &client.ObjectKey{Namespace: fields[2], Name: fields[3]}
	}

	return nil
}

func (l *remote) match(fields []string, kind schema.GroupKind) bool {
	switch len(fields) {
	case 4:
		if fields[0] == kind.Group && fields[1] == kind.Kind && l.cluster == "" {
			return true
		}
	case 5:
		if fields[0] == kind.Group && fields[1] == kind.Kind && fields[4] == l.cluster {
			return true
		}
	}
	return false
}

func (l *remote) SetOwner(owner client.Object, obj client.Object) error {
	gvk, err := apiutil.GVKForObject(owner, l.scheme)

	if err != nil {
		return err
	}
	if gvk.Group == "" {
		gvk.Group = "core"
	}
	o := fmt.Sprintf("%s/%s/%s/%s", gvk.Group, gvk.Kind, owner.GetNamespace(), owner.GetName())
	if l.cluster != "" {
		o += "/" + l.cluster
	}
	objutils.SetAnnotation(obj, l.property, o)
	return nil
}
