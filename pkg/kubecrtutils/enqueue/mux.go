package enqueue

import (
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type Mux interface {
	TriggerSource(obj runtime.Object) (Enqueue, error)

	EnqueueByGVK(gvk schema.GroupVersionKind, key client.ObjectKey)
	EnqueueByObject(obj runtime.Object) error
}

type mux struct {
	lock     sync.Mutex
	scheme   *runtime.Scheme
	enqueues map[schema.GroupVersionKind]Enqueue
}

func NewMux(scheme *runtime.Scheme) Mux {
	return &mux{scheme: scheme, enqueues: make(map[schema.GroupVersionKind]Enqueue)}
}

func (m *mux) TriggerSource(obj runtime.Object) (Enqueue, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	gvk, err := apiutil.GVKForObject(obj, m.scheme)
	if err != nil {
		return nil, err
	}
	e := m.enqueues[gvk]
	if e == nil {
		e = NewEnqueue()
		m.enqueues[gvk] = e
	}
	return e, nil
}

func (m *mux) EnqueueByGVK(gvk schema.GroupVersionKind, key client.ObjectKey) {
	m.lock.Lock()
	defer m.lock.Unlock()

	e := m.enqueues[gvk]
	if e != nil {
		e.AddToQueue(key)
	}
}

func (m *mux) EnqueueByObject(obj runtime.Object) error {
	var err error
	gvk := obj.GetObjectKind().GroupVersionKind()
	if gvk.Empty() {
		gvk, err = apiutil.GVKForObject(obj, m.scheme)
		if err != nil {
			return err
		}
	}
	k, err := GetKey(obj)
	if err != nil {
		return err
	}
	m.EnqueueByGVK(gvk, k)
	return nil
}

func GetKey(obj runtime.Object) (client.ObjectKey, error) {
	// runtime.Object is an interface that doesn't strictly guarantee
	// access to metadata. client.Object adds those methods.
	accessor, ok := obj.(client.Object)
	if !ok {
		return client.ObjectKey{}, fmt.Errorf("object does not implement client.Object")
	}

	return client.ObjectKey{
		Name:      accessor.GetName(),
		Namespace: accessor.GetNamespace(),
	}, nil
}
