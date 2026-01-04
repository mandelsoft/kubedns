package kubecrtutils

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ObjectPointer[T any] interface {
	client.Object
	*T
}
