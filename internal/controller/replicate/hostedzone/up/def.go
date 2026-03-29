package up

import (
	"context"
	"sync"

	"github.com/mandelsoft/kubecrtutils/cacheindex"
	"github.com/mandelsoft/kubecrtutils/cluster"
	"github.com/mandelsoft/kubecrtutils/controller"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconcile"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/common"
	"github.com/mandelsoft/kubedns/internal/controller/replicate"
	common2 "github.com/mandelsoft/kubedns/internal/controller/replicate/common"
	"github.com/mandelsoft/kubedns/internal/controller/replicate/common/up"
	"github.com/mandelsoft/logging"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	mcreconcile "sigs.k8s.io/multicluster-runtime/pkg/reconcile"
)

func Controller() controller.Definition {
	return up.Controller[*corednsv1alpha1.HostedZone, corednsv1alpha1.HostedZone](
		replicate.ControllerHostedzone,
		replicate.HOSTEDZONE_GROUP,
		common2.ProviderFunc(GetMapping),
		NewHandler,
	).
		AddIndex(replicate.IndexKeyZoneParent, parentIndexer).
		AddForeignIndex(cacheindex.Define[*corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry](replicate.IndexKeyEntryZone, replicate.SOURCE, zoneIndexer))
}

func zoneIndexer(res *corednsv1alpha1.CoreDNSEntry) []string {
	if res.Spec.ZoneRef == "" {
		return nil
	}
	return []string{res.Spec.ZoneRef}
}

func parentIndexer(o *corednsv1alpha1.HostedZone) []string {
	if o.Spec.ParentRef == "" {
		return nil
	}
	return []string{o.Spec.ParentRef}
}

////////////////////////////////////////////////////////////////////////////////

type Handler struct {
	lock          sync.Mutex
	unresponsible map[mcreconcile.Request]bool

	ParentIndex cacheindex.TypedIndex[corednsv1alpha1.HostedZone]
	EntryIndex  cacheindex.TypedIndex[corednsv1alpha1.CoreDNSEntry]

	Source cluster.ClusterEquivalent
}

func NewHandler(c controller.TypedController[*corednsv1alpha1.HostedZone, corednsv1alpha1.HostedZone]) (up.ResponsibilityHandler[*corednsv1alpha1.HostedZone, corednsv1alpha1.HostedZone], error) {
	pidx, err := cacheindex.GetIndexFrom[corednsv1alpha1.HostedZone](c, replicate.IndexKeyZoneParent)
	if err != nil {
		return nil, err
	}
	eidx, err := cacheindex.GetIndexFrom[corednsv1alpha1.CoreDNSEntry](c, replicate.IndexKeyEntryZone)
	if err != nil {
		return nil, err
	}

	return &Handler{
		unresponsible: map[mcreconcile.Request]bool{},
		ParentIndex:   pidx,
		EntryIndex:    eidx,
		Source:        c.GetCluster(),
	}, nil
}

func GetMapping(o *common2.Options) common2.Mapping {
	return o.Zones
}

func (h *Handler) SetStatusCondition(obj *corednsv1alpha1.HostedZone, condition metav1.Condition) bool {
	if condition.ObservedGeneration == 0 {
		condition.ObservedGeneration = obj.GetGeneration()
	}
	return meta.SetStatusCondition(&obj.Status.Conditions, condition)
}

func (h *Handler) SetResponsibility(r *up.ReconcileRequest[*corednsv1alpha1.HostedZone, corednsv1alpha1.HostedZone], obj *corednsv1alpha1.HostedZone) {
	if obj.Spec.ParentRef == "" {
		if r.Reconciler.Options.TargetClass != "" {
			obj.Spec.Class = &r.Reconciler.Options.TargetClass
		} else {
			obj.Spec.Class = nil
		}
	}
}

func (h *Handler) Delete(r *up.ReconcileRequest[*corednsv1alpha1.HostedZone, corednsv1alpha1.HostedZone]) {
	delete(h.unresponsible, r.Request)
	key := r.Request.NamespacedName
	h.TriggerChildren(r, r, key)
	h.TriggerEntries(r, r, key)
}

func (h *Handler) IsResponsible(r *up.ReconcileRequest[*corednsv1alpha1.HostedZone, corednsv1alpha1.HostedZone]) (bool, reconcile.Problem) {
	info, ok, prob := common.GetRootInfo(r, r, r, r.Object)

	if prob != nil {
		if !ok {
			// temporary problem, potentially responsible
			r.Info("temporary problem", "problem", prob)
			return true, prob
		}
		// config problem, cannot determine responsibility
		r.Info("cannot determine responsibility", "problem", prob)
		h.SetStatusCondition(r.Object, metav1.Condition{
			Type:    corednsv1alpha1.ValidationConditionType,
			Status:  metav1.ConditionFalse, // Use metav1 constant
			Reason:  corednsv1alpha1.ReasonInvalidParent,
			Message: prob.Error().Error(),
		})
		r.Object.Status.Message = prob.Error().Error()
		r.Object.Status.State = "Problem"
		// replicate anyway
		r.Info("replicate for unknown responsibility")
		h.Handle(r, true)
		return true, nil
	}

	if info.Class == r.Reconciler.Options.Class {
		h.Handle(r, true)
		return true, nil
	}
	r.Info("not responsible: found class \"{{class}}\", but expected \"{{expected}}\"", "class", info.Class, "expected", r.Reconciler.Options.Class)
	h.Handle(r, false)
	return false, nil
}

func (h *Handler) Handle(r *up.ReconcileRequest[*corednsv1alpha1.HostedZone, corednsv1alpha1.HostedZone], resp bool) {
	h.lock.Lock()
	defer h.lock.Unlock()

	if b, ok := h.unresponsible[r.Request]; ok && b == resp {
		// unchanged -> do nothing
		return
	}

	r.Info("responsibility changed -> trigger dependent")
	h.unresponsible[r.Request] = resp
	// responsibility has been changed
	// trigger children + entries.
	key := r.Request.NamespacedName
	h.TriggerChildren(r, r, key)
	h.TriggerEntries(r, r, key)
}

func (r *Handler) GetNestedZones(ctx context.Context, ns string, n string) ([]corednsv1alpha1.HostedZone, error) {
	return r.ParentIndex.GetTyped(ctx, ns, n)
}

func (r *Handler) GetEntriesForZone(ctx context.Context, ns string, n string) ([]corednsv1alpha1.CoreDNSEntry, error) {
	return r.EntryIndex.GetTyped(ctx, ns, n)
}

func (r *Handler) TriggerChildren(ctx context.Context, logger logging.Logger, obj client.ObjectKey) error {
	logger.Info("notify children about changes")
	children, err := r.GetNestedZones(ctx, obj.Namespace, obj.Name)
	if err != nil {
		return err
	}
	for _, c := range children {
		logger.Info("triggering child", "name", c.Name, "namespace", c.Namespace)
		r.Source.EnqueueByObject(ctx, &c)
	}
	return nil
}

func (r *Handler) TriggerEntries(ctx context.Context, logger logging.Logger, obj client.ObjectKey) error {
	entries, err := r.GetEntriesForZone(ctx, obj.Namespace, obj.Name)
	if err != nil {
		return err
	}
	logger.Info("notify {{amount}} entries about changes", "amount", len(entries))
	for _, c := range entries {
		r.Source.EnqueueByObject(ctx, &c)
	}
	return nil
}
