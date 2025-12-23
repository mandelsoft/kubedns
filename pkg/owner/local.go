package owner

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type local struct {
	scheme *runtime.Scheme
}

func LocalOwner(scheme *runtime.Scheme) OwnerHandler {
	return &local{scheme}
}

func (l *local) GetOwner(obj client.Object, kind schema.GroupKind) *client.ObjectKey {
	for _, r := range obj.GetOwnerReferences() {
		if r.Kind == kind.Kind {}
		gv, _ := schema.ParseGroupVersion(r.APIVersion)
		if gv.Group == kind.Group {
			return &client.ObjectKey{Name: r.Name, Namespace: obj.GetNamespace()}
		}
	}
	return nil
}

func (l *local)	SetOwner(owner client.Object, obj client.Object) error {
	return controllerutil.SetControllerReference(owner, obj, l.scheme)
}
