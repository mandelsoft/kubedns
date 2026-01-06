package internal

import (
	"errors"
	"fmt"
	"maps"
)

type Group[T Named] interface {
	Get(name string) T
	Add(elem ...T) error
	Elements(yield func(string, T) bool)
	Len() int
}

type _group[T Named] struct {
	Mutex
	typename string
	elements map[string]T
}

func NewGroup[T Named](name string) Group[T] {
	return &_group[T]{typename: name, elements: make(map[string]T)}
}

func newGroup[T Named](name string) _group[T] {
	return _group[T]{typename: name, elements: make(map[string]T)}
}

func (c *_group[T]) GetName() string {
	return c.typename
}

func (c *_group[T]) Len() int {
	defer c.Lock()()
	return len(c.elements)
}

func (c *_group[T]) Add(elem ...T) error {
	var err error
	defer c.Lock()()

	for _, e := range elem {
		if _, ok := c.elements[e.GetName()]; ok {
			err = errors.Join(err, fmt.Errorf("%s %q already exists", c.typename, e.GetName()))
		} else {
			c.elements[e.GetName()] = e
		}
	}
	return err
}

func (c *_group[T]) Get(name string) T {
	defer c.Lock()()
	return c.elements[name]
}

func (c *_group[T]) Elements(yield func(string, T) bool) {
	c.Mutex.Lock()
	m := maps.Clone(c.elements)
	c.Mutex.Unlock()
	for n, elem := range m {
		if !yield(n, elem) {
			return
		}
	}
}
