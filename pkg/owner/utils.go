package owner

import (
	"context"
	"reflect"

	"github.com/mandelsoft/goutils/generics"
	"github.com/mandelsoft/kubedns/pkg/clusterutils"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func MapOwnerToLocalRequestByObject[O, P client.Object](owner OwnerHandler, p clusterutils.SchemeProvider, obj P) handler.TypedMapFunc[O, reconcile.Request] {
	gk, err := clusterutils.GKForObject(p, obj)
	if err != nil {
		panic(err)
	}
	return MapOwnerToLocalRequest[O](owner, gk)
}

func MapOwnerToLocalRequest[O client.Object](owner OwnerHandler, kind schema.GroupKind) handler.TypedMapFunc[O, reconcile.Request] {
	return func(ctx context.Context, obj O) []reconcile.Request {
		key := owner.GetOwner(obj, kind)
		if key == nil {
			return nil
		}
		return []reconcile.Request{
			{
				NamespacedName: *key,
			},
		}
	}
}

func WatchSourceForSlave[O, R client.Object](c clusterutils.Cluster, owner OwnerHandler, s clusterutils.SchemeProvider) source.Source {
	// O,R are pointer types, but we need an object

	o := reflect.New(generics.TypeOf[O]().Elem()).Interface().(O)
	r := reflect.New(generics.TypeOf[R]().Elem()).Interface().(R)

	return source.Kind(c.GetCache(), o,
		handler.TypedEnqueueRequestsFromMapFunc[O, reconcile.Request](MapOwnerToLocalRequestByObject[O](owner, s, r)))

}

func AddOwnerModifier(handler OwnerHandler, owner client.Object) clusterutils.ObjectModifier {
	return func(obj client.Object) error { return handler.SetOwner(owner, obj) }
}
