package entry

import (
	"context"

	"github.com/mandelsoft/flagutils"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/common"
	"github.com/mandelsoft/kubedns/pkg/options/manageropts"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Create(opts flagutils.OptionSetProvider) error {
	Log.Info("creating hostedzone controller...")
	mgmt := manageropts.GetManagementContext(opts)

	r := &CoreDNSEntryReconciler{
		Reconciler: common.NewReconciler(Log, mgmt.Get("dataplane"), "coredns.mandelsoft.org/coreendentry"),
	}
	r.Logger = Log
	r.Info("using dataplane cluster", "apiserver", r.DataPlane.GetConfig().Host)

	if err := r.DataPlane.GetFieldIndexer().IndexField(context.Background(), &corednsv1alpha1.CoreDNSEntry{}, common.IndexKeyEntryZone, func(rawObj client.Object) []string {
		res := rawObj.(*corednsv1alpha1.CoreDNSEntry)
		if res.Spec.ZoneRef == "" {
			return nil
		}
		return []string{res.Spec.ZoneRef}
	}); err != nil {
		return err
	}

	return mgmt.NewController().
		For(&corednsv1alpha1.CoreDNSEntry{}).
		Named("coredns-corednsentry").
		Complete(r)
}
