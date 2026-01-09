package common

import (
	"context"
	"fmt"

	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cacheindex"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cluster"
	controller2 "github.com/mandelsoft/kubedns/pkg/kubecrtutils/controller"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/types"
	"github.com/mandelsoft/logging"
)

const ControllerHostedzone = "hostedzone"
const IndexKeyZoneParent = "hostedzone.parent"

const ControllerEntry = "corednsentry"
const IndexKeyEntryZone = "corednsentry.zone"

type Reconciler struct {
	logging.Logger
	FieldManager string
	DataPlane    cluster.Cluster

	ParentIndex cacheindex.Index[corednsv1alpha1.HostedZone]
	EntryIndex  cacheindex.Index[corednsv1alpha1.CoreDNSEntry]
}

func NewReconciler(controller types.Controller) (*Reconciler, error) {
	i := controller.GetControllerManager().GetIndices()

	pidx := i.Get(controller2.GlobalControllerIndexName(ControllerHostedzone, IndexKeyZoneParent))
	if pidx == nil {
		return nil, fmt.Errorf("parent index not found")
	}
	eidx := i.Get(controller2.GlobalControllerIndexName(ControllerEntry, IndexKeyEntryZone))
	if eidx == nil {
		return nil, fmt.Errorf("entry index not found")
	}
	return &Reconciler{
		Logger:       controller.GetLogger(),
		FieldManager: controller.GetFieldManager(),
		DataPlane:    controller.GetCluster(),
		ParentIndex:  pidx.(cacheindex.Index[corednsv1alpha1.HostedZone]),
		EntryIndex:   eidx.(cacheindex.Index[corednsv1alpha1.CoreDNSEntry]),
	}, nil
}

func (r *Reconciler) GetNestedZones(ctx context.Context, ns string, n string) ([]corednsv1alpha1.HostedZone, error) {
	return r.ParentIndex.GetTyped(ctx, ns, n)
}

func (r *Reconciler) GetEntriesForZone(ctx context.Context, ns string, n string) ([]corednsv1alpha1.CoreDNSEntry, error) {
	return r.EntryIndex.GetTyped(ctx, ns, n)
}
