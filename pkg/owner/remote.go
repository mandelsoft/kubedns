package owner

import (
	"fmt"
	"strings"

	"github.com/mandelsoft/kubedns/pkg/objutils"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

const DEFAULT_REMOTE_NAME = "cross-cluster.io/owner-id"

type remote struct {
	scheme *runtime.Scheme
	name   string
	class  string
}

func RemoteOwner(scheme *runtime.Scheme, name string, class string) OwnerHandler {
	if name == "" {
		name = DEFAULT_REMOTE_NAME
	}
	return &remote{scheme: scheme, class: class, name: name}
}

func (l *remote) GetOwner(obj client.Object, kind schema.GroupKind) *client.ObjectKey {

	// group / kind / namespace / name / class
	a := objutils.GetAnnotation(obj, l.name)
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
		if fields[0] == kind.Group && fields[1] == kind.Kind && l.class == "" {
			return true
		}
	case 5:
		if fields[0] == kind.Group && fields[1] == kind.Kind && fields[4] == l.class {
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
	if l.class == "" {
		o += "/" + l.class
	}
	objutils.SetAnnotation(obj, l.name, o)
	return nil
}
