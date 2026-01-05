package cluster

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/sets"
)

type Mappings map[string]string

func IdMapping(set sets.Set[string]) Mappings {
	m := Mappings{}
	for n := range set {
		m[n] = n
	}
	return m
}

func Map(clusters Clusters, maps Mappings) (Clusters, error) {
	n := NewClusters()

	for local, global := range maps {
		c := clusters.Get(global)
		if c == nil {
			return nil, fmt.Errorf("global cluster %q for %q not defined", global, local)
		}
		n.Add(NewAlias(local, c))
	}
	return n, nil
}
