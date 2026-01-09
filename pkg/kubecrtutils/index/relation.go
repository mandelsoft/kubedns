package index

import (
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type relation struct {
	usersFor *base
	usedBy   *base
}

func newRelation() *relation {
	return &relation{usersFor: newBase(), usedBy: newBase()}
}

func (r *relation) Add(a, b client.ObjectKey) bool {
	r.usersFor.Add(b, a)
	return r.usedBy.Add(a, b)
}

func (r *relation) Replace(a client.ObjectKey, b ...client.ObjectKey) bool {
	old := r.usedBy.Get(a)
	exp := sets.New(b...)
	removed := old.Difference(exp)
	added := exp.Difference(old)

	for c := range added {
		r.usersFor.Add(c, a)
		r.usedBy.Add(a, c)
	}
	for c := range removed {
		r.usersFor.Remove(c, a)
		r.usedBy.Remove(a, c)
	}
	return old.Len() != 0 || exp.Len() != 0
}

func (r *relation) Remove(a, b client.ObjectKey) bool {
	if r == nil {
		return false
	}
	r.usersFor.Remove(b, a)
	return r.usedBy.Remove(a, b)
}

func (r *relation) RemoveObject(o client.ObjectKey) bool {
	if r == nil {
		return false
	}
	r.usersFor.DeleteObject(o)
	return r.usedBy.DeleteObject(o)
}

func (r *relation) UsedBy(a client.ObjectKey) sets.Set[client.ObjectKey] {
	if r == nil {
		return nil
	}
	return r.usedBy.Get(a)
}

func (r *relation) UsersFor(b client.ObjectKey) sets.Set[client.ObjectKey] {
	if r == nil {
		return nil
	}
	return r.usersFor.Get(b)
}

func (r *relation) Clear() bool {
	if r == nil {
		return false
	}
	r.usersFor.Clear()
	return r.usedBy.Clear()
}
