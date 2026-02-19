package common

import (
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconciler"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ReconcilationRequest[T client.Object, R any] struct {
	reconciler.BaseRequest[T]
	Reconciler R
}
