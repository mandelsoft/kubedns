/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package entry

import (
	"context"

	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/common"
	"github.com/mandelsoft/kubedns/internal/controller/hostedzone"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/controller"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/controller/controllerutils/reconciler"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	crtreconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// +kubebuilder:rbac:groups=coredns.mandelsoft.org,resources=corednsentries,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coredns.mandelsoft.org,resources=corednsentries/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coredns.mandelsoft.org,resources=corednsentries/finalizers,verbs=update

// CoreDNSEntryReconciler reconciles a CoreDNSEntry object.
// This is done by creating a reconciliation request, which
// then executes the reconciliation tasks.
type CoreDNSEntryReconciler struct {
	*common.Reconciler
	Options *hostedzone.Options
}

func (r *CoreDNSEntryReconciler) Request(def *reconciler.BaseRequest[*corednsv1alpha1.CoreDNSEntry]) reconciler.ReconcileRequest[*corednsv1alpha1.CoreDNSEntry] {
	return &ReconcileRequest{
		reconciler.DefaultReconcileRequest[*corednsv1alpha1.CoreDNSEntry, *CoreDNSEntryReconciler]{*def, r},
	}
}

func CreateReconciler(ctx context.Context, controller controller.Controller[corednsv1alpha1.CoreDNSEntry, *corednsv1alpha1.CoreDNSEntry], b *builder.Builder) (crtreconcile.Reconciler, error) {
	base, err := common.NewReconciler(controller)
	if err != nil {
		return nil, err
	}
	base.Info("creating entry reconciler...")

	d := controller.GetControllerManager().GetControllerDefinition(common.ControllerHostedzone)

	r := &CoreDNSEntryReconciler{
		Reconciler: base,
		Options:    &d.GetOptions().(*hostedzone.ReconcilerFactory).Options,
	}
	r.Info("using dataplane cluster", "apiserver", r.DataPlane.GetConfig().Host)
	return reconciler.CRTReconcilerFor[*corednsv1alpha1.CoreDNSEntry](controller, r, 0), nil
}
