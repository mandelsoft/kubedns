package index

import (
	"context"
	"fmt"
	"reflect"

	"github.com/mandelsoft/kubedns/pkg/kubecrtutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cluster"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/internal"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type IndexerFunc[T client.Object] = func(T) []string

type Definition interface {
	GetName() string
	GetCluster() string
	GetIndexerFunc() client.IndexerFunc
	Apply(ctx context.Context, set cluster.Clusters) (types.Index, error)
}

type TypedDefinition[T any, P kubecrtutils.ObjectPointer[T]] interface {
	Definition
	TypedApply(ctx context.Context, set types.Clusters) (Index[T], error)
}

type _definition[T any, P kubecrtutils.ObjectPointer[T]] struct {
	internal.Element
	cluster string
	proto   client.Object
	idxfunc IndexerFunc[P]
}

func NewDefinition[T any, P kubecrtutils.ObjectPointer[T]](name string, cluster string, idxfunc IndexerFunc[P]) TypedDefinition[T, P] {
	tType := reflect.TypeOf((P)(nil)).Elem()
	baseType := tType.Elem()

	proto := reflect.New(baseType).Interface().(P)

	return &_definition[T, P]{
		Element: internal.NewElement(name),
		cluster: cluster,
		idxfunc: idxfunc,
		proto:   proto,
	}
}

func (d *_definition[T, P]) GetCluster() string {
	return d.cluster
}

func (d *_definition[T, P]) GetIndexerFunc() client.IndexerFunc {
	return d.indexer
}

func (d *_definition[T, P]) indexer(obj client.Object) []string {
	return d.idxfunc(obj.(any).(P))

}

func (d *_definition[T, P]) Apply(ctx context.Context, set cluster.Clusters) (cluster.Index, error) {
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
			c,
		}, nil
	})

	if err != nil {
		return nil, err
	}
	return idx, nil
}

func (d *_definition[T, P]) TypedApply(ctx context.Context, set cluster.Clusters) (Index[T], error) {
	idx, err := d.Apply(ctx, set)
	if err != nil {
		return nil, err
	}
	return idx.(Index[T]), nil
}
