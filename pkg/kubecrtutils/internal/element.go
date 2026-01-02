package internal

type Named interface {
	GetName() string
}

type Element struct {
	name string
}

var _ Named = (*Element)(nil)

func NewElement(name string) Element {
	return Element{name}
}

func (e *Element) GetName() string {
	return e.name
}
