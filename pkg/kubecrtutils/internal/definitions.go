package internal

import (
	"fmt"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/goutils/errors"
	"github.com/spf13/pflag"
)

type Definitions[T Named, D any] interface {
	flagutils.Options
	flagutils.OptionSetProvider
	Add(elem ...T) D
	Get(name string) T
	GetError() error
	Elements(yield func(string, T) bool)
}

type DefinitionsImpl[T Named, D any] struct {
	_group[T]
	self    D
	options flagutils.DefaultOptionSet
	errlist *errors.ErrorList
}

var _ Definitions[Named, any] = (*DefinitionsImpl[Named, any])(nil)

func NewDefinitions[T Named, D any](typ string, self D) DefinitionsImpl[T, D] {
	return DefinitionsImpl[T, D]{
		_group:  newGroup[T](typ),
		self:    self,
		errlist: errors.ErrListf("%s definitions", typ),
	}
}

func (d *DefinitionsImpl[T, D]) GetTypeName() string {
	return d.typename
}

func (d *DefinitionsImpl[T, D]) Add(elems ...T) D {
	defer d.Lock()()

	for _, e := range elems {
		if _, ok := d.elements[e.GetName()]; ok {
			d.errlist.Add(fmt.Errorf("duplicate %s %s definition", d.typename, e.GetName()))
		} else {
			flagutils.AddOptionally(&d.options, e)
			d.elements[e.GetName()] = e
		}
	}
	return d.self
}

func (d *DefinitionsImpl[T, D]) GetError() error {
	defer d.Lock()()
	return d.errlist.Result()
}

////////////////////////////////////////////////////////////////////////////////

func (d *DefinitionsImpl[T, D]) AddFlags(fs *pflag.FlagSet) {
	d.options.AddFlags(fs)
}

func (d *DefinitionsImpl[T, D]) AsOptionSet() flagutils.OptionSet {
	return d.options
}
