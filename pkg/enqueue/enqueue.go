package enqueue

import (
	"context"
	"sync"

	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type Enqueue interface {
	source.Source
	AddToQueue(key client.ObjectKey)
}

type enqueue struct {
	lock   sync.Mutex
	queues []workqueue.TypedRateLimitingInterface[reconcile.Request]
}

var _ Enqueue= (*enqueue)(nil)

func NewEnqueue() Enqueue {
	return &enqueue{}
}

func (e *enqueue) Start(ctx context.Context, w workqueue.TypedRateLimitingInterface[reconcile.Request]) error {
	e.lock.Lock()
	defer e.lock.Unlock()
	for _, q := range e.queues {
		if w == q {
			return nil
		}
	}
	e.queues = append(e.queues, w)
	return nil
}

////////////////////////////////////////////////////////////////////////////////

func (e *enqueue) AddToController(c *ctrl.Builder) *ctrl.Builder {
	c.WatchesRawSource(e)
	return c
}

////////////////////////////////////////////////////////////////////////////////

func (e *enqueue) AddToQueue(key client.ObjectKey) {
	e.lock.Lock()
	defer e.lock.Unlock()

	for _, q := range e.queues {
		q.AddRateLimited(reconcile.Request{NamespacedName: key})
	}
}
