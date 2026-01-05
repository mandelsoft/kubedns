package index

import (
	"sync"

	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type UntypedIndex interface {
	Replace(r string, a client.ObjectKey, b ...client.ObjectKey) bool
	Add(r string, a, b client.ObjectKey) bool
	Remove(r string, a, b client.ObjectKey) bool
	RemoveObject(o client.ObjectKey) bool
	UsersFor(r string, b client.ObjectKey) sets.Set[client.ObjectKey]
	UsedBy(r string, a client.ObjectKey) sets.Set[client.ObjectKey]
	Clear(r string) bool
}

type untyped struct {
	lock      sync.Mutex
	relations map[string]*relation
}

func NewUntyped() UntypedIndex {
	return &untyped{
		relations: make(map[string]*relation),
	}
}

func (u *untyped) Add(r string, a, b client.ObjectKey) bool {
	u.lock.Lock()
	defer u.lock.Unlock()

	return u.getRelation(r).Add(a, b)
}

func (u *untyped) Replace(r string, a client.ObjectKey, b ...client.ObjectKey) bool {
	u.lock.Lock()
	defer u.lock.Unlock()

	return u.getRelation(r).Replace(a, b...)
}

func (u *untyped) Remove(r string, a, b client.ObjectKey) bool {
	u.lock.Lock()
	defer u.lock.Unlock()

	return u.relations[r].Remove(a, b)
}

func (u *untyped) RemoveObject(o client.ObjectKey) bool {
	u.lock.Lock()
	defer u.lock.Unlock()

	ok := false
	for _, rel := range u.relations {
		ok = rel.RemoveObject(o) || ok
	}
	return ok
}

func (u *untyped) UsedBy(r string, a client.ObjectKey) sets.Set[client.ObjectKey] {
	u.lock.Lock()
	defer u.lock.Unlock()

	return u.relations[r].UsedBy(a)
}

func (u *untyped) UsersFor(r string, b client.ObjectKey) sets.Set[client.ObjectKey] {
	u.lock.Lock()
	defer u.lock.Unlock()

	return u.relations[r].UsersFor(b)

}

func (u *untyped) Clear(r string) bool {
	u.lock.Lock()
	defer u.lock.Unlock()

	return u.relations[r].Clear()
}

func (u *untyped) getRelation(r string) *relation {
	rel := u.relations[r]
	if rel == nil {
		rel = newRelation()
		u.relations[r] = rel
	}
	return rel
}
