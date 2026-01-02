package cluster

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Lister func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error

type index struct {
	name        string
	cluster     Cluster
	listFactory func() (client.ObjectList, error)
}

func NewDefaultIndex(name string, cluster Cluster, proto runtime.Object) (Index, error) {
	fac, err := createListFactoryFromObject(cluster.GetScheme(), proto)
	if err != nil {
		return nil, err
	}
	return &index{
		name:        name,
		cluster:     cluster,
		listFactory: fac,
	}, nil
}

func (i *index) GetName() string {
	return i.name
}

func (i *index) GetCluster() Cluster {
	return i.cluster
}

func (i *index) GetList(ctx context.Context, namespace, key string) (client.ObjectList, error) {
	list, err := i.listFactory()
	if err != nil {
		return nil, err
	}
	err = i.cluster.List(ctx, list, client.InNamespace(namespace), client.MatchingFields{i.name: key})
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (i *index) ForEachListItem(ctx context.Context, namespace, key string, action func(object runtime.Object) error) error {
	list, err := i.GetList(ctx, namespace, key)
	if err != nil {
		return err
	}
	return meta.EachListItem(list, action)
}

func (i *index) Trigger(ctx context.Context, namespace, key string) error {
	return i.ForEachListItem(ctx, namespace, key, i.cluster.EnqueueByObject)
}

func createListFromObject(scheme *runtime.Scheme, obj runtime.Object) (client.ObjectList, error) {
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

func createListFactoryFromObject(scheme *runtime.Scheme, obj runtime.Object) (func() (client.ObjectList, error), error) {
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
