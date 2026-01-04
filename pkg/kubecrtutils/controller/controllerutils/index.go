package controllerutils

import (
	"context"
	"fmt"

	"github.com/mandelsoft/goutils/sliceutils"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/types"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Indexer[T any, P kubecrtutils.ObjectPointer[T]] func(obj P) []string

type Index[T any, P kubecrtutils.ObjectPointer[T]] interface {
}

type _index[T any, P kubecrtutils.ObjectPointer[T]] struct {
	cluster     types.Cluster
	name        string
	listFactory func() (client.ObjectList, error)
}

func NewIndex[T any, P kubecrtutils.ObjectPointer[T]](cluster types.Cluster, name string, indexer Indexer[T, P]) (Index[T, P], error) {
	if err := cluster.GetFieldIndexer().IndexField(context.Background(), &corednsv1alpha1.CoreDNSEntry{}, name, func(rawObj client.Object) []string {
		res := rawObj.(P)
		return indexer(res)
	}); err != nil {
		return nil, err
	}
	var proto T
	fac, err := CreateListFactoryFromObject(cluster.GetScheme(), any(&proto).(runtime.Object))
	if err != nil {
		return nil, err
	}
	return &_index[T, P]{cluster: cluster, name: name, listFactory: fac}, nil
}

func (i *_index[T, P]) Get(ctx context.Context, namespace, key string) ([]T, error) {
	list, err := i.listFactory()
	if err != nil {
		return nil, err
	}
	err = i.cluster.List(ctx, list, client.InNamespace(namespace), client.MatchingFields{i.name: key})
	if err != nil {
		return nil, err
	}

	items, err := meta.ExtractList(list)
	if err != nil {
		return nil, err
	}
	return sliceutils.Convert[T](items), nil
}

func CreateListFromObject(scheme *runtime.Scheme, obj runtime.Object) (client.ObjectList, error) {
	// 1. Get the GVK for the prototype object
	gvks, _, err := scheme.ObjectKinds(obj)
	if err != nil || len(gvks) == 0 {
		return nil, fmt.Errorf("could not determine GVK for object: %w", err)
	}

	// 2. Create the List GVK (e.g., "HostedZone" -> "HostedZoneList")
	gvk := gvks[0]
	listGVK := schema.GroupVersionKind{
		Group:   gvk.Group,
		Version: gvk.Version,
		Kind:    gvk.Kind + "List",
	}

	// 3. Ask the scheme to instantiate the List type
	listObj, err := scheme.New(listGVK)
	if err != nil {
		return nil, fmt.Errorf("could not create list type %s: %w", listGVK.Kind, err)
	}

	return listObj.(client.ObjectList), nil
}

func CreateListFactoryFromObject(scheme *runtime.Scheme, obj runtime.Object) (func() (client.ObjectList, error), error) {
	// 1. Get the GVK for the prototype object
	gvks, _, err := scheme.ObjectKinds(obj)
	if err != nil || len(gvks) == 0 {
		return nil, fmt.Errorf("could not determine GVK for object: %w", err)
	}

	// 2. Create the List GVK (e.g., "HostedZone" -> "HostedZoneList")
	gvk := gvks[0]
	listGVK := schema.GroupVersionKind{
		Group:   gvk.Group,
		Version: gvk.Version,
		Kind:    gvk.Kind + "List",
	}

	// 3. Ask the scheme to instantiate the List type
	_, err = scheme.New(listGVK)
	if err != nil {
		return nil, fmt.Errorf("could not create list type %s: %w", listGVK.Kind, err)
	}

	return func() (client.ObjectList, error) {
		l, err := scheme.New(listGVK)
		if err != nil {
			return nil, err
		}
		return l.(client.ObjectList), nil
	}, nil
}
