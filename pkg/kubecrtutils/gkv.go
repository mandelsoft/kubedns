package kubecrtutils

import (
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cluster"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func GKVForObject(c cluster.SchemeProvider, obj client.Object) (schema.GroupVersionKind, error) {
	return apiutil.GVKForObject(obj, c.GetScheme())
}

func GKForObject(c cluster.SchemeProvider, obj client.Object) (schema.GroupKind, error) {
	gkv, err := apiutil.GVKForObject(obj, c.GetScheme())
	if err != nil {
		return schema.GroupKind{}, err
	}
	return schema.GroupKind{Group: gkv.Group, Kind: gkv.Kind}, nil
}
