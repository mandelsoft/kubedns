package owner

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type local struct {
	remote OwnerHandler
	scheme *runtime.Scheme
}

func LocalOwner(scheme *runtime.Scheme, remote OwnerHandler) OwnerHandler {
	if remote == nil {
		remote = RemotePropertyOwner(scheme, "")
	}
	return &local{remote, scheme}
}

func (l *local) GetOwner(obj client.Object, kind schema.GroupKind) *client.ObjectKey {
	// Step 1: check cluster and namespace local references
	for _, r := range obj.GetOwnerReferences() {
		if r.Kind != kind.Kind {
			continue
		}
		gv, _ := schema.ParseGroupVersion(r.APIVersion)
		if gv.Group == kind.Group || (kind.Group == "core" && gv.Group == "") {
			return &client.ObjectKey{Name: r.Name, Namespace: obj.GetNamespace()}
		}
	}
	// Step 2; check cross namespace and cross cluster references
	if l.remote != nil {
		return l.remote.GetOwner(obj, kind)
	}
	return nil
}

func (l *local) SetOwner(owner client.Object, obj client.Object) error {
	if owner.GetNamespace() != obj.GetNamespace() {
		return l.remote.SetOwner(owner, obj)
	}
	return controllerutil.SetControllerReference(owner, obj, l.scheme)
}
