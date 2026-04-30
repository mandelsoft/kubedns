package servercomp

import (
	"sync"

	"github.com/mandelsoft/kubecrtutils/ctrlmgmt"
	"github.com/mandelsoft/kubecrtutils/types"
	mcreconcile "sigs.k8s.io/multicluster-runtime/pkg/reconcile"
)

type Serial struct {
	lock  sync.Mutex
	zones map[mcreconcile.Request]int
}

func GetSerial(mgr types.ControllerManager) *Serial {
	return ctrlmgmt.GetData(mgr, SerialKey, newSerial)
}

func newSerial() *Serial {
	return &Serial{zones: make(map[mcreconcile.Request]int)}
}

func (s *Serial) Add(key mcreconcile.Request, id int) {
	s.lock.Lock()
	defer s.lock.Unlock()

	old := s.zones[key]
	if id > old {
		s.zones[key] = id
	}
}

func (s *Serial) Get(key mcreconcile.Request) int {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.zones[key]
}
