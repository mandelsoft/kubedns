package common

import (
	"context"

	"github.com/mandelsoft/kubecrtutils/cacheindex"
	"github.com/mandelsoft/kubecrtutils/cluster"
	"github.com/mandelsoft/kubecrtutils/controller"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/common"
	"github.com/mandelsoft/logging"
)

const ControllerHostedzone = "hostedzone"
const IndexKeyZoneParent = common.IndexKeyZoneParent

const ControllerEntry = "corednsentry"
const IndexKeyEntryZone = common.IndexKeyEntryZone

var ParentIndexer = common.ParentIndexer
var ZoneIndexer = common.ZoneIndexer

type Reconciler struct {
	logging.Logger
	FieldManager string

	Dataplane   cluster.ClusterEquivalent
	ParentIndex cacheindex.TypedIndex[corednsv1alpha1.HostedZone]
	EntryIndex  cacheindex.TypedIndex[corednsv1alpha1.CoreDNSEntry]
}

func NewReconciler(c controller.Controller) (*Reconciler, error) {
	pidx, err := cacheindex.GetIndexFrom[corednsv1alpha1.HostedZone](c, IndexKeyZoneParent)
	if err != nil {
		return nil, err
	}
	eidx, err := cacheindex.GetIndexFrom[corednsv1alpha1.CoreDNSEntry](c, IndexKeyEntryZone)
	if err != nil {
		return nil, err
	}

	return &Reconciler{
		Logger:       c.GetLogger(),
		FieldManager: c.GetFieldManager(),
		Dataplane:    c.GetCluster(),
		ParentIndex:  pidx,
		EntryIndex:   eidx,
	}, nil
}

func (r *Reconciler) GetNestedZones(ctx context.Context, ns string, n string) ([]corednsv1alpha1.HostedZone, error) {
	return r.ParentIndex.GetTyped(ctx, ns, n)
}

func (r *Reconciler) GetEntriesForZone(ctx context.Context, ns string, n string) ([]corednsv1alpha1.CoreDNSEntry, error) {
	return r.EntryIndex.GetTyped(ctx, ns, n)
}
