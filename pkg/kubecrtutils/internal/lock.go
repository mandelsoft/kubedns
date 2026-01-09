package internal

import (
	"sync"
)

type Mutex struct {
	sync.Mutex
}

func (m *Mutex) Lock() func() {
	m.Mutex.Lock()
	return m.Unlock
}
