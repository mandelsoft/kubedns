package entry

import (
	"context"

	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/common"
	"github.com/mandelsoft/kubedns/internal/controller/hostedzone"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/controller"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func Controller() controller.Definition {
	return controller.DefineByFunc[corednsv1alpha1.CoreDNSEntry](common.ControllerEntry, "dataplane", CreateReconciler).
		UseCluster("runtime").
		AddIndex(common.IndexKeyEntryZone, zoneIndexer)
}

func zoneIndexer(res *corednsv1alpha1.CoreDNSEntry) []string {
	if res.Spec.ZoneRef == "" {
		return nil
	}
	return []string{res.Spec.ZoneRef}
}

////////////////////////////////////////////////////////////////////////////////

func CreateReconciler(ctx context.Context, controller controller.Controller[corednsv1alpha1.CoreDNSEntry, *corednsv1alpha1.CoreDNSEntry], b *builder.Builder) (reconcile.Reconciler, error) {
	logger := controller.GetLogger()

	base, err := common.NewReconciler(controller)
	if err != nil {
		return nil, err
	}
	logger.Info("creating entry reconciler...")

	d := controller.GetControllerManager().GetControllerDefinition(common.ControllerHostedzone)

	r := &CoreDNSEntryReconciler{
		Reconciler: base,
		Options:    &d.GetOptions().(*hostedzone.ReconcilerFactory).Options,
	}
	r.Info("using dataplane cluster", "apiserver", r.DataPlane.GetConfig().Host)
	return r, nil
}
