package owner

import (
	"context"
	"reflect"

	"github.com/mandelsoft/goutils/generics"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils"
	clusterutils2 "github.com/mandelsoft/kubedns/pkg/kubecrtutils/cluster"
	"github.com/mandelsoft/logging"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func MapOwnerToLocalRequestByObject[O, P client.Object](owner OwnerHandler, p clusterutils2.SchemeProvider, obj P, log ...logging.Logger) handler.TypedMapFunc[O, reconcile.Request] {
	gk, err := kubecrtutils.GKForObject(p, obj)
	if err != nil {
		panic(err)
	}
	return MapOwnerToLocalRequest[O](owner, gk, log...)
}

func MapOwnerToLocalRequest[O client.Object](owner OwnerHandler, kind schema.GroupKind, log ...logging.Logger) handler.TypedMapFunc[O, reconcile.Request] {
	return func(ctx context.Context, obj O) []reconcile.Request {
		key := owner.GetOwner(obj, kind)
		if key == nil {
			return nil
		}
		if len(log) > 0 {
			log[0].Info("trigger owner {{owner}} of modified object {{modified}}",
				"owner", *key,
				"modified", client.ObjectKeyFromObject(obj))
		}
		return []reconcile.Request{
			{
				NamespacedName: *key,
			},
		}
	}
}

func WatchSourceForSlave[O, R client.Object](c clusterutils2.Cluster, owner OwnerHandler, s clusterutils2.SchemeProvider, log ...logging.Logger) source.Source {
	// O,R are pointer types, but we need an object

	o := reflect.New(generics.TypeOf[O]().Elem()).Interface().(O)
	r := reflect.New(generics.TypeOf[R]().Elem()).Interface().(R)

	return source.Kind(c.GetCache(), o,
		handler.TypedEnqueueRequestsFromMapFunc[O, reconcile.Request](MapOwnerToLocalRequestByObject[O](owner, s, r, log...)))

}

func AddOwnerModifier(handler OwnerHandler, owner client.Object) clusterutils2.ObjectModifier {
	return func(obj client.Object) error { return handler.SetOwner(owner, obj) }
}
