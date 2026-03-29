package common

import (
	"context"
	"sync"

	"github.com/mandelsoft/kubecrtutils/cluster/cluster"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconcile"
	"github.com/mandelsoft/logging"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	mcreconcile "sigs.k8s.io/multicluster-runtime/pkg/reconcile"
)

type MappingProvider interface {
	GetMapping(*Options) Mapping
}

type ProviderFunc func(*Options) Mapping

func (p ProviderFunc) GetMapping(o *Options) Mapping {
	return p(o)
}

type Context interface {
	context.Context
	cluster.Cluster
	logging.Logger
}

type base interface {
	context.Context
	logging.Logger
}

type _context struct {
	// omit cluster view
	base
	cluster.Cluster
}

func WithCluster(ctx Context, cl cluster.Cluster) Context {
	return &_context{
		ctx, cl,
	}
}

type Mapping interface {
	GetOriginal(key client.ObjectKey) *mcreconcile.Request
	SetOriginal(ctx Context, key client.ObjectKey, tgt mcreconcile.Request) reconcile.Problem
	RemoveOriginal(ctx Context, key client.ObjectKey) reconcile.Problem
}

type _mapping struct {
	lock   *sync.RWMutex
	own    *ResourceIndex
	others []*ResourceIndex
}

func NewDefaultMapping(own *ResourceIndex, lock *sync.RWMutex, others ...*ResourceIndex) Mapping {
	if lock == nil {
		lock = new(sync.RWMutex)
	}
	return &_mapping{
		lock:   lock,
		own:    own,
		others: others,
	}
}

func (m *_mapping) GetOriginal(key client.ObjectKey) *mcreconcile.Request {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.own.get(key)
}

func (v *_mapping) SetOriginal(ctx Context, key client.ObjectKey, tgt mcreconcile.Request) reconcile.Problem {
	v.lock.Lock()
	defer v.lock.Unlock()
	return v.own.add(ctx, key, tgt)
}

func (v *_mapping) RemoveOriginal(ctx Context, key client.ObjectKey) reconcile.Problem {
	v.lock.Lock()
	defer v.lock.Unlock()

	empty := true
	for _, other := range v.others {
		if len(other.entries[key.Namespace]) != 0 {
			empty = false
			break
		}
	}
	return v.own.remove(empty, ctx, key)
}

type ResourceIndex struct {
	entries map[string]map[string]mcreconcile.Request
}

func NewResourceIndex() *ResourceIndex {
	return &ResourceIndex{
		entries: map[string]map[string]mcreconcile.Request{},
	}
}

func (i *ResourceIndex) assureNamespace(ctx Context, namespace string) reconcile.Problem {
	var ns v1.Namespace
	err := ctx.Get(ctx, client.ObjectKey{Name: namespace}, &ns)
	if err != nil {
		if errors.IsNotFound(err) {
			ctx.Info("assure target namespace", "namespace", namespace)
			ns.Name = namespace
			err = ctx.Create(ctx, &ns)
		}
	}
	return reconcile.TemporaryProblem(err)
}

func (i *ResourceIndex) get(key client.ObjectKey) *mcreconcile.Request {
	obs := i.entries[key.Namespace]
	if len(obs) == 0 {
		return nil
	}
	r, ok := obs[key.Name]
	if !ok {
		return nil
	}
	return &r
}

func (i *ResourceIndex) add(ctx Context, key client.ObjectKey, tgt mcreconcile.Request) reconcile.Problem {
	p := i.assureNamespace(ctx, key.Namespace)
	if p != nil {
		return p
	}
	obs := i.entries[key.Namespace]
	if obs == nil {
		obs = make(map[string]mcreconcile.Request)
		i.entries[key.Namespace] = obs
	}
	obs[key.Name] = tgt
	return nil
}

func (i *ResourceIndex) remove(empty bool, ctx Context, key client.ObjectKey) reconcile.Problem {
	obs := i.entries[key.Namespace]
	if empty && lastEntry(obs, key.Name) {
		ctx.Info("cleanup target namespace {{targetnamespace}}", "targetnamespace", key.Namespace)
		ns := v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: key.Namespace,
			},
		}

		err := ctx.Delete(ctx, &ns)
		if err != nil && !errors.IsNotFound(err) {
			return reconcile.TemporaryProblem(err)
		}
	} else {
		s := ""
		if !empty {
			s = " and other mappings not empty"
		}
		no := len(obs)
		if no > 0 {
			if _, ok := obs[key.Name]; ok {
				no--
			}
		}
		ctx.Info("still {{no}} mappings in target namespace {{targetnamespace}}"+s, "no", no, "targetnamespace", key.Namespace)
	}

	if obs != nil {
		delete(obs, key.Name)
		if len(obs) == 0 {
			delete(i.entries, key.Namespace)
		}
	}
	return nil
}

func lastEntry[T any](m map[string]T, key string) bool {
	if len(m) != 1 {
		return len(m) == 0
	}
	for k := range m {
		if k == key {
			return true
		}
	}
	return false
}
