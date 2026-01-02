package index

import (
	"context"

	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cluster"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Index[T any] interface {
	cluster.Index

	Get(namespace string, name string) ([]T, error)
	ForEachItem(ctx context.Context, namespace, key string, action func(object *T) error) error
}

type _index[T any] struct {
	cluster.Index
	name    string
	cluster cluster.Cluster
}

func (i *_index[T]) Get(ctx context.Context, namespace string, name string) ([]T, error) {
	list, err := i.GetList(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	return GetItemList[T](list)
}

func (i *_index[T]) ForEachItem(ctx context.Context, namespace, key string, action func(object *T) error) error {
	list, err := i.Get(ctx, namespace, key)
	if err != nil {
		return err
	}
	for _, e := range list {
		err := action(&e)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetItemList[T any](list client.ObjectList) ([]T, error) {
	ptr, err := meta.GetItemsPtr(list)
	if err != nil {
		return nil, err
	}
	return *(ptr).(*[]T), nil
}
