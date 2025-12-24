package index

import (
	"maps"
	"sync"

	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type UntypedIndex interface {
	Add(r string, a, b client.ObjectKey) bool
	Delete(r string, a, b client.ObjectKey) bool
	DeleteObject(o client.ObjectKey)
	GetUsers(r string, b client.ObjectKey) sets.Set[client.ObjectKey]
	Clear(r string) bool
}

type untyped struct {
	lock  sync.Mutex
	index map[string]map[client.ObjectKey]sets.Set[client.ObjectKey]
}

func NewUntyped() UntypedIndex {
	return &untyped{index: make(map[string]map[client.ObjectKey]sets.Set[client.ObjectKey])}
}

func (u *untyped) Add(r string, a, b client.ObjectKey) bool {
	u.lock.Lock()
	defer u.lock.Unlock()

	rel := u.getRelation(r)

	t := rel[b]
	if t == nil {
		t = make(sets.Set[client.ObjectKey], 0)
		rel[b] = t
	}
	if t.Has(a) {
		return false
	}
	t.Insert(a)
	return true
}

func (u *untyped) Delete(r string, a, b client.ObjectKey) bool {
	u.lock.Lock()
	defer u.lock.Unlock()

	rel := u.index[r]
	if rel == nil {
		return false
	}

	t := rel[b]
	if t == nil {
		return false
	}
	if t.Has(a) {
		t.Delete(a)
		return true
	}
	return false
}

func (u *untyped) DeleteObject(o client.ObjectKey) {
	u.lock.Lock()
	defer u.lock.Unlock()

	for r, rel := range u.index {
		delete(rel, o)
		if len(rel) == 0 {
			delete(u.index, r)
			continue
		}
		for b, t := range rel {
			t.Delete(o)
			if len(t) == 0 {
				delete(rel, b)
			}
		}
	}
}

func (u *untyped) GetUsers(r string, b client.ObjectKey) sets.Set[client.ObjectKey] {
	u.lock.Lock()
	defer u.lock.Unlock()

	rel := u.index[r]
	if rel == nil {
		return nil
	}

	t := rel[b]
	if t == nil {
		return nil
	}
	return maps.Clone(t)
}

func (u *untyped) Clear(r string) bool {
	u.lock.Lock()
	defer u.lock.Unlock()

	if u.index[r] == nil {
		return false
	}
	delete(u.index, r)
	return true
}

func (u *untyped) getRelation(r string) map[client.ObjectKey]sets.Set[client.ObjectKey] {
	rel := u.index[r]
	if rel == nil {
		rel = make(map[client.ObjectKey]sets.Set[client.ObjectKey])
		u.index[r] = rel
	}
	return rel
}
