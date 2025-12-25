package index

import (
	"maps"

	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type base struct {
	index map[client.ObjectKey]sets.Set[client.ObjectKey]
}

func newBase() *base {
	return &base{index: make(map[client.ObjectKey]sets.Set[client.ObjectKey])}
}

func (u *base) Add(a, b client.ObjectKey) bool {

	t := u.index[a]
	if t == nil {
		t = make(sets.Set[client.ObjectKey], 0)
		u.index[a] = t
	}
	if t.Has(b) {
		return false
	}
	t.Insert(b)
	return true
}

func (u *base) Remove(a, b client.ObjectKey) bool {
	t := u.index[a]
	if t == nil {
		return false
	}
	if t.Has(b) {
		t.Delete(b)
		return true
	}
	return false
}

func (u *base) DeleteObject(o client.ObjectKey) bool {
	_, ok := u.index[o]
	if ok {
		delete(u.index, o)
	}
	for b, t := range u.index {
		old := len(t)
		t.Delete(o)
		ok = ok || old != len(t)
		if len(t) == 0 {
			delete(u.index, b)
		}
	}
	return ok
}

func (u *base) Get(a client.ObjectKey) sets.Set[client.ObjectKey] {
	t := u.index[a]
	if t == nil {
		return nil
	}
	return maps.Clone(t)
}

func (u *base) Clear() bool {
	ok := len(u.index) != 0
	clear(u.index)
	return ok
}
