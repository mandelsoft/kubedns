package common

import (
	"sync"
)

type ReplicationMapping struct {
	Entries Mapping
	Zones   Mapping
}

func NewReplicationMapping() *ReplicationMapping {
	lock := new(sync.RWMutex)
	entries := NewResourceIndex()
	zones := NewResourceIndex()

	return &ReplicationMapping{
		Entries: NewDefaultMapping(entries, lock, zones),
		Zones:   NewDefaultMapping(zones, lock, entries),
	}
}
