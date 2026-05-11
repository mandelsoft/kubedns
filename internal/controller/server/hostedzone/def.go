package hostedzone

import (
	"context"

	"github.com/mandelsoft/kubecrtutils/cacheindex"
	"github.com/mandelsoft/kubecrtutils/controller"
	"github.com/mandelsoft/kubecrtutils/controller/constraints"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconciler"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconciler/factories"
	"github.com/mandelsoft/kubecrtutils/types"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/common"
	"github.com/mandelsoft/kubedns/internal/controller/server"
	"github.com/mandelsoft/kubedns/internal/controller/server/servercomp"
	"github.com/mandelsoft/kubedns/internal/controller/server/zonemodel"
	"github.com/mandelsoft/logging"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Controller() controller.Definition {
	return controller.Define[*corednsv1alpha1.HostedZone, corednsv1alpha1.HostedZone](server.ControllerHostedzone, server.CLUSTER,
		factories.NewByFactory[*server.Options, *Settings, *corednsv1alpha1.HostedZone, corednsv1alpha1.HostedZone](&Factory{}),
	).
		AddIndex(common.IndexKeyZoneParent, common.ParentIndexer).
		AddForeignIndex(cacheindex.Define[*corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry](common.IndexKeyEntryZone, server.CLUSTER, common.ZoneIndexer)).
		InGroup(server.GROUP).UseComponent(server.Component).
		WithActivationConstraint(constraints.Complete(server.GROUP))
}

type Settings struct {
	model *zonemodel.Model
	*server.Options
	types.ClusterEquivalent
	ParentIndex cacheindex.TypedIndex[corednsv1alpha1.HostedZone]
	EntryIndex  cacheindex.TypedIndex[corednsv1alpha1.CoreDNSEntry]
}

type Factory struct {
	factories.DefaultFactory[factories.None, *Settings, *corednsv1alpha1.HostedZone, corednsv1alpha1.HostedZone]
}

func (f *Factory) CreateOptions() *server.Options {
	return server.NewOptions()
}

func (f *Factory) CreateSettings(ctx context.Context, o *server.Options, controller controller.TypedController[*corednsv1alpha1.HostedZone, corednsv1alpha1.HostedZone]) (*Settings, error) {
	return &Settings{
		model:             controller.GetComponents().Get(server.Component).GetImplementation().(*servercomp.Component).GetModel(),
		Options:           o,
		ClusterEquivalent: controller.GetMainCluster(),
		EntryIndex:        cacheindex.GetTypedIndex[corednsv1alpha1.CoreDNSEntry](controller.GetIndices(), common.IndexKeyEntryZone),
		ParentIndex:       cacheindex.GetTypedIndex[corednsv1alpha1.HostedZone](controller.GetIndices(), common.IndexKeyZoneParent),
	}, nil
}

func (f *Factory) CreateRequest(r *reconciler.BaseRequest[*corednsv1alpha1.HostedZone], r2 *factories.Reconciler[*server.Options, *Settings, *corednsv1alpha1.HostedZone, corednsv1alpha1.HostedZone]) reconciler.ReconcileRequest[*corednsv1alpha1.HostedZone] {
	var cond []v1.Condition
	if r.Object != nil {
		cond = r.Object.Status.Conditions
	}
	return &Request{r, r2.Settings, r2.Options.IsMaster(cond)}
}

////////////////////////////////////////////////////////////////////////////////

func (r *Settings) GetNestedZones(ctx context.Context, ns string, n string) ([]corednsv1alpha1.HostedZone, error) {
	return r.ParentIndex.GetTyped(ctx, ns, n)
}

func (r *Settings) GetEntriesForZone(ctx context.Context, ns string, n string) ([]corednsv1alpha1.CoreDNSEntry, error) {
	return r.EntryIndex.GetTyped(ctx, ns, n)
}

func (r *Settings) TriggerChildren(ctx context.Context, logger logging.Logger, obj client.ObjectKey) error {
	logger.Info("notify children about changes")
	children, err := r.GetNestedZones(ctx, obj.Namespace, obj.Name)
	if err != nil {
		return err
	}
	for _, c := range children {
		logger.Info("triggering child", "name", c.Name, "namespace", c.Namespace)
		r.EnqueueByObject(ctx, &c)
	}
	return nil
}

func (r *Settings) TriggerEntries(ctx context.Context, logger logging.Logger, obj client.ObjectKey) error {
	entries, err := r.GetEntriesForZone(ctx, obj.Namespace, obj.Name)
	if err != nil {
		return err
	}
	logger.Info("notify {{amount}} children about changes", "amount", len(entries))
	for _, c := range entries {
		r.EnqueueByObject(ctx, &c)
	}
	return nil
}
