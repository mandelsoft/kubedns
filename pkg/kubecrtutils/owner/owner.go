package owner

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type OwnerHandler interface {
	GetOwner(obj client.Object, kind schema.GroupKind) *client.ObjectKey
	SetOwner(owner client.Object, obj client.Object) error
}
