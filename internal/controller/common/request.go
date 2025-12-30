package common

import (
	"context"

	"github.com/mandelsoft/logging"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ReconcilationContext struct {
	context.Context
	logging.Logger
	client.ObjectKey
}

type ReconcilationRequest[T, R any] struct {
	ReconcilationContext
	Instance   T
	Reconciler R
}
