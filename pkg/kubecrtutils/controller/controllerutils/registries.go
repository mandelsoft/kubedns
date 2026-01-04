package controllerutils

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"golang.org/x/exp/maps"
)

// Factory acts a factory for a target object using a configuration.
type Factory[C any, O any] interface {
	Create(ctx context.Context, cfg C) (O, error)
	Description() string
}

type FactoryFunc[C any, O any] func(context.Context, C) (O, error)

func (f FactoryFunc[C, O]) Create(ctx context.Context, cfg C) (O, error) {
	return f(ctx, cfg)
}

func (f FactoryFunc[C, O]) Description() string {
	return ""
}

// Registry registers a set of named factory alternative for a particular purpose.
// It can then be used to create an appropriate instace for a given type name and configuration.
type Registry[C any, O any] interface {
	Register(name string, factory Factory[C, O])
	Names() []string

	Get(name string) Factory[C, O]
	Create(ctx context.Context, name string, cfg C) (O, error)
}

type registry[C, O any] struct {
	lock      sync.RWMutex
	typ       string
	factories map[string]Factory[C, O]
}

func NewRegistry[C any, O any](typ string) Registry[C, O] {
	return &registry[C, O]{typ: typ, factories: make(map[string]Factory[C, O])}
}

func (r *registry[C, O]) Register(name string, f Factory[C, O]) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.factories[name] = f
}

func (r *registry[C, O]) Names() []string {
	r.lock.Lock()
	defer r.lock.Unlock()
	keys := maps.Keys(r.factories)
	sort.Strings(keys)
	return keys
}

func (r *registry[C, O]) Get(name string) Factory[C, O] {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.factories[name]
}

func (r *registry[C, O]) Create(ctx context.Context, name string, cfg C) (O, error) {
	var _nil O

	r.lock.Lock()
	defer r.lock.Unlock()

	f := r.factories[name]
	if f == nil {
		return _nil, fmt.Errorf("unknown %s type %q", r.typ, name)
	}
	return f.Create(ctx, cfg)
}
