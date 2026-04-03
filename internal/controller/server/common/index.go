package common

import (
	"context"
	"fmt"

	"github.com/mandelsoft/kubecrtutils/cacheindex"
	"github.com/mandelsoft/kubecrtutils/cluster"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/server/zonemodel"
)

type Index struct {
	Cluster    cluster.ClusterEquivalent
	EntryIndex cacheindex.TypedIndex[corednsv1alpha1.CoreDNSEntry]
}

var _ zonemodel.Index = (*Index)(nil)

func (i *Index) LookupRelativeDomainName(ctx context.Context, zone zonemodel.ZoneKey, rdn string) ([]corednsv1alpha1.CoreDNSEntry, error) {
	return i.EntryIndex.GetTyped(ctx, zone.Namespace, EntryKey(zone, rdn))
}

func (i *Index) GetZone(ctx context.Context, zone zonemodel.ZoneKey) (*corednsv1alpha1.HostedZone, error) {
	var obj corednsv1alpha1.HostedZone
	err := i.Cluster.Get(ctx, zone.NamespacedName, &obj)
	if err != nil {
		return nil, err
	}
	return &obj, nil
}

func EntryKey(zone zonemodel.ZoneKey, rdn string) string {
	return fmt.Sprintf("%s/%s/%s", zone.Namespace, zone.Name, rdn)
}
