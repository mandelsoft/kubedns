package index

import (
	"sync"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TypedObjectKey struct {
	schema.GroupKind
	client.ObjectKey
}

func NewTypedObjectKey(groupKind schema.GroupKind, objectkey client.ObjectKey) TypedObjectKey {
	return TypedObjectKey{GroupKind: groupKind, ObjectKey: objectkey}
}

func NewTypedObjectKeyByNames(groupKind schema.GroupKind, namespace, name string) TypedObjectKey {
	return TypedObjectKey{GroupKind: groupKind, ObjectKey: client.ObjectKey{Namespace: namespace, Name: name}}
}

type TypedIndex interface {
	Replace(r string, a TypedObjectKey, b ...TypedObjectKey) bool
	Add(r string, a, b TypedObjectKey) bool
	Remove(r string, a, b TypedObjectKey) bool
	RemoveObject(o TypedObjectKey) bool
	UsersForKind(r string, b TypedObjectKey, gk schema.GroupKind) sets.Set[client.ObjectKey]
	UsersFor(r string, b TypedObjectKey) sets.Set[TypedObjectKey]
	UsedByKind(r string, a TypedObjectKey, gk schema.GroupKind) sets.Set[client.ObjectKey]
	UsedBy(r string, a TypedObjectKey) sets.Set[TypedObjectKey]
	Clear(r string) bool
}

type typed struct {
	lock      sync.Mutex
	relations map[string]map[schema.GroupKind]map[schema.GroupKind]*relation
}

func NewTyped() TypedIndex {
	return &typed{
		relations: make(map[string]map[schema.GroupKind]map[schema.GroupKind]*relation),
	}
}

func (u *typed) Add(r string, a, b TypedObjectKey) bool {
	u.lock.Lock()
	defer u.lock.Unlock()

	return u.getRelationFor(r, a.GroupKind, b.GroupKind).Add(a.ObjectKey, b.ObjectKey)
}

func (u *typed) Replace(r string, a TypedObjectKey, b ...TypedObjectKey) bool {
	u.lock.Lock()
	defer u.lock.Unlock()

	urel := u.getUserRelationFor(r, a.GroupKind)

	targets := map[schema.GroupKind][]client.ObjectKey{}

	for _, c := range b {
		targets[c.GroupKind] = append(targets[c.GroupKind], c.ObjectKey)
	}

	ok := false
	for gk, c := range targets {
		rel := urel[gk]
		if rel == nil {
			rel = newRelation()
			urel[gk] = rel
		}
		ok = rel.Replace(a.ObjectKey, c...) || ok
	}
	return ok
}

func (u *typed) Remove(r string, a, b TypedObjectKey) bool {
	u.lock.Lock()
	defer u.lock.Unlock()

	return u.getRelationFor(r, a.GroupKind, b.GroupKind).Remove(a.ObjectKey, b.ObjectKey)
}

func (u *typed) RemoveObject(o TypedObjectKey) bool {
	u.lock.Lock()
	defer u.lock.Unlock()

	ok := false
	for _, r := range u.relations {
		for _, users := range r[o.GroupKind] {
			ok = users.RemoveObject(o.ObjectKey) || ok
		}
	}
	return ok
}

func (u *typed) UsedByKind(r string, a TypedObjectKey, gk schema.GroupKind) sets.Set[client.ObjectKey] {
	u.lock.Lock()
	defer u.lock.Unlock()

	return u.queryRelationFor(r, a.GroupKind, gk).UsedBy(a.ObjectKey)
}

func (u *typed) UsersForKind(r string, b TypedObjectKey, gk schema.GroupKind) sets.Set[client.ObjectKey] {
	u.lock.Lock()
	defer u.lock.Unlock()

	return u.queryRelationFor(r, gk, b.GroupKind).UsersFor(b.ObjectKey)

}

func (u *typed) UsedBy(r string, a TypedObjectKey) sets.Set[TypedObjectKey] {
	u.lock.Lock()
	defer u.lock.Unlock()

	urel := u.queryUserRelationFor(r, a.GroupKind)
	if len(urel) == 0 {
		return nil
	}

	result := sets.New[TypedObjectKey]()
	for gk, rel := range urel {
		s := rel.UsedBy(a.ObjectKey)
		for u := range s {
			result.Insert(TypedObjectKey{GroupKind: gk, ObjectKey: u})
		}
	}
	return result
}

func (u *typed) UsersFor(r string, b TypedObjectKey) sets.Set[TypedObjectKey] {
	u.lock.Lock()
	defer u.lock.Unlock()

	urel := u.relations[r]
	if urel == nil {
		return nil
	}
	result := sets.New[TypedObjectKey]()
	for gk, used := range urel {
		rel := used[b.GroupKind]
		if rel != nil {
			for u := range rel.UsersFor(b.ObjectKey) {
				result.Insert(TypedObjectKey{GroupKind: gk, ObjectKey: u})
			}
		}
	}
	return result

}

func (u *typed) Clear(r string) bool {
	u.lock.Lock()
	defer u.lock.Unlock()

	rel := u.relations[r]
	if rel == nil || len(rel) == 0 {
		delete(u.relations, r)
		return false
	}
	delete(u.relations, r)
	return true
}

func (u *typed) getRelation(r string) map[schema.GroupKind]map[schema.GroupKind]*relation {
	rel := u.relations[r]
	if rel == nil {
		rel = make(map[schema.GroupKind]map[schema.GroupKind]*relation)
		u.relations[r] = rel
	}
	return rel
}

func (u *typed) getUserRelationFor(r string, users schema.GroupKind) map[schema.GroupKind]*relation {
	rel := u.getRelation(r)

	urel := rel[users]
	if urel == nil {
		urel = make(map[schema.GroupKind]*relation)
		rel[users] = urel
	}
	return urel
}

func (u *typed) queryUserRelationFor(r string, users schema.GroupKind) map[schema.GroupKind]*relation {
	rel := u.relations[r]
	if rel == nil {
		return nil
	}
	return rel[users]
}

func (u *typed) getRelationFor(r string, users schema.GroupKind, used schema.GroupKind) *relation {
	rel := u.getUserRelationFor(r, users)

	urel := rel[used]
	if urel == nil {
		urel = newRelation()
		rel[used] = urel
	}
	return urel
}

func (u *typed) queryRelationFor(r string, users schema.GroupKind, used schema.GroupKind) *relation {
	rel := u.queryUserRelationFor(r, users)
	if rel == nil {
		return nil
	}
	return rel[used]
}
