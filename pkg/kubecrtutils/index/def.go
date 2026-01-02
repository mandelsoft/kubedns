package index

import (
	"context"
	"fmt"
	"reflect"

	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cluster"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/internal"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ObjectPointer[T any] interface {
	client.Object
	*T
}

type IndexerFunc[T client.Object] = func(T) []string

type Definition interface {
	GetName() string
	GetCluster() string
	GetIndexerFunc() client.IndexerFunc
	Apply(ctx context.Context, set cluster.Clusters) (cluster.Index, error)
}

type DefinitionT[T any] interface {
	Definition
	ApplyT(ctx context.Context, set cluster.Clusters) (Index[T], error)
}

type _definition[P ObjectPointer[T], T any] struct {
	internal.Element
	cluster string
	proto   client.Object
	idxfunc IndexerFunc[P]
}

func NewDefinitionT[P ObjectPointer[T], T any](name string, cluster string, idxfunc IndexerFunc[P]) DefinitionT[T] {
	tType := reflect.TypeOf((P)(nil)).Elem()
	baseType := tType.Elem()

	proto := reflect.New(baseType).Interface().(P)

	return &_definition[P, T]{
		Element: internal.NewElement(name),
		cluster: cluster,
		idxfunc: idxfunc,
		proto:   proto,
	}
}

func (d *_definition[P, T]) GetCluster() string {
	return d.cluster
}

func (d *_definition[P, T]) GetIndexerFunc() client.IndexerFunc {
	return d.indexer
}

func (d *_definition[P, T]) indexer(obj client.Object) []string {
	return d.idxfunc(obj.(any).(P))

}

func (d *_definition[P, T]) Apply(ctx context.Context, set cluster.Clusters) (cluster.Index, error) {
	c := set.Get(d.GetName())
	if c == nil {
		return nil, fmt.Errorf("cluster %q not found", d.GetName())
	}

	idx, err := c.CreateIndex(ctx, d.GetName(), d.proto, d.indexer, func(_c cluster.Cluster, name string) (cluster.Index, error) {
		idx, err := cluster.NewDefaultIndex(d.GetName(), _c, d.proto)
		if err != nil {
			return nil, err
		}
		return &_index[T]{
			idx,
			d.GetName(),
			c,
		}, nil
	})

	if err != nil {
		return nil, err
	}
	return idx, nil
}

func (d *_definition[P, T]) ApplyT(ctx context.Context, set cluster.Clusters) (Index[T], error) {
	idx, err := d.Apply(ctx, set)
	if err != nil {
		return nil, err
	}
	return idx.(Index[T]), nil
}
