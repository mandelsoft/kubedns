package objutils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Filter interface {
	Filter(obj client.Object) bool
}

type FilterFunc func(obj client.Object) bool

func (f FilterFunc) Filter(obj client.Object) bool {
	return f(obj)
}

func MetaGroupKindFilter(gk metav1.GroupKind) Filter {
	return GroupKindFilter(gk.Group, gk.Kind)
}

func SchemaGroupKindFilter(gk schema.GroupKind) Filter {
	return GroupKindFilter(gk.Group, gk.Kind)
}

func GroupKindFilter(group, kind string) Filter {
	return FilterFunc(func(obj client.Object) bool {
		gvk := obj.GetObjectKind().GroupVersionKind()
		if gvk.Group == "core" {
			gvk.Group = ""
		}
		if group == "core" {
			group = ""
		}
		return gvk.Group == group && gvk.Kind == kind
	})
}

func Or(filters ...Filter) Filter {
	return FilterFunc(func(obj client.Object) bool {
		for _, f := range filters {
			if f.Filter(obj) {
				return true
			}
		}
		return false
	})
}

func And(filters ...Filter) Filter {
	return FilterFunc(func(obj client.Object) bool {
		for _, f := range filters {
			if !f.Filter(obj) {
				return false
			}
		}
		return true
	})
}

func Not(filter Filter) Filter {
	return FilterFunc(func(obj client.Object) bool {
		return !filter.Filter(obj)
	})
}
