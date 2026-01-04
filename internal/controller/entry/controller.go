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
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/controller/controllerutils/reconcile"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

// CoreDNSEntryReconciler reconciles a CoreDNSEntry object
type CoreDNSEntryReconciler struct {
	*common.Reconciler
	Options *hostedzone.Options
}

// +kubebuilder:rbac:groups=coredns.mandelsoft.org,resources=corednsentries,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coredns.mandelsoft.org,resources=corednsentries/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coredns.mandelsoft.org,resources=corednsentries/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CoreDNSEntry object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.4/pkg/reconcile
func (r *CoreDNSEntryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := Log.WithName(req.String()).WithValues("object", req.NamespacedName)

	log.Info("Reconciling CoreDNSEntry")
	var obj corednsv1alpha1.CoreDNSEntry
	err := r.DataPlane.Get(ctx, req.NamespacedName, &obj)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("entry has been deleted")
			err = nil
		}
		return reconcile.Result(log, reconcile.TemporaryProblem(err))
	}

	prob := NewRequest(ctx, r, log, &obj).handleObject()
	return reconcile.Result(log, prob)
}
