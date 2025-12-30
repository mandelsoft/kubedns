package common

import (
	"context"

	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/pkg/clusterutils"
	"github.com/mandelsoft/logging"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const IndexKeyZoneParent = "hostedzone.parent"
const IndexKeyEntryZone = "corednsentry.zone"

type Reconciler struct {
	logging.Logger
	FieldManager string
	DataPlane    clusterutils.Cluster
}

func NewReconciler(logger logging.Logger, dataPlane clusterutils.Cluster, fieldManager string) *Reconciler {
	return &Reconciler{
		Logger:       logger,
		FieldManager: fieldManager,
		DataPlane:    dataPlane,
	}
}

func (r *Reconciler) GetNestedZones(ctx context.Context, ns string, n string) []corednsv1alpha1.HostedZone {
	var list corednsv1alpha1.HostedZoneList

	err := r.DataPlane.List(ctx, &list, client.InNamespace(ns), client.MatchingFields{IndexKeyZoneParent: n})
	if err != nil {
		return nil
	}

	return list.Items
}

func (r *Reconciler) GetEntriesForZone(ctx context.Context, ns string, n string) []corednsv1alpha1.CoreDNSEntry {
	var list corednsv1alpha1.CoreDNSEntryList

	err := r.DataPlane.List(ctx, &list, client.InNamespace(ns), client.MatchingFields{IndexKeyEntryZone: n})
	if err != nil {
		return nil
	}

	return list.Items
}
