package owner

import (
	"github.com/mandelsoft/kubedns/pkg/objutils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type cond struct {
	OwnerHandler
	filter objutils.Filter
}

func Conditional(o OwnerHandler, f objutils.Filter) OwnerHandler {
	return &cond{o, f}
}

func (c *cond) SetOwner(owner client.Object, obj client.Object) error {
	if c.Filter(obj) {
		return c.OwnerHandler.SetOwner(owner, obj)
	}
	return nil
}

func (c *cond) Filter(obj client.Object) bool {
	if c.filter == nil {
		return true
	}
	return c.filter.Filter(obj)
}
