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

package hostedzone

import (
	"context"

	"github.com/mandelsoft/kubecrtutils/controller"
	reconcile2 "github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconciler"
	"github.com/mandelsoft/kubecrtutils/types"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const INDEX_SASECFRET = "serviceaccount-secret"

func Controller() controller.Definition {
	return controller.Define[*corednsv1alpha1.HostedZone](common.ControllerHostedzone, "dataplane", &ReconcilerFactory{}).
		UseCluster("runtime").
		AddIndex(common.IndexKeyZoneParent, parentIndexer).
		AddTrigger(
			controller.OwnerTrigger[*appsv1.Deployment]().OnCluster("runtime"),
			controller.OwnerTrigger[*corev1.Secret]().OnCluster("runtime"),
			controller.LocalResourceTriggerByFactory[*corev1.Secret](secretTriggerFactory),
		)
}

func parentIndexer(o *corednsv1alpha1.HostedZone) []string {
	if o.Spec.ParentRef == "" {
		return nil
	}
	return []string{o.Spec.ParentRef}
}

func secretTriggerFactory(ctx context.Context, cntr types.Controller) handler.TypedMapFunc[*corev1.Secret, reconcile.Request] {
	r := cntr.GetReconciler().(reconcile2.CRTReconciler).GetEffective().(*HostedZoneReconciler)
	log := cntr.GetLogger()
	return func(ctx context.Context, obj *corev1.Secret) []reconcile.Request {
		var trigger []reconcile.Request
		key := client.ObjectKeyFromObject(obj)
		users := r.index.UsersFor(INDEX_SASECFRET, key)
		if len(users) > 0 {
			log.Info("change of service account secret {{secret}} triggers {{amount}} zones",
				"secret", key,
				"amount", len(users))
		}
		for user := range users {
			zone := apitypes.NamespacedName{
				Name:      user.Name,
				Namespace: user.Namespace,
			}
			trigger = append(trigger,
				reconcile.Request{
					NamespacedName: zone,
				},
			)
		}
		return trigger
	}
}
