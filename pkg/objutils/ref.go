package objutils

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RefObjectKeyFor(object client.Object, name string) client.ObjectKey {
	return client.ObjectKey{Namespace: object.GetNamespace(), Name: name}
}

func RefIndexKeyFor(object client.Object, name string) string {
	return client.ObjectKey{Namespace: object.GetNamespace(), Name: name}.String()
}
