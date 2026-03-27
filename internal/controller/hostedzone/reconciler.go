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

	"github.com/mandelsoft/kubecrtutils/cluster"
	. "github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconcile"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconciler"
	"github.com/mandelsoft/kubecrtutils/index"
	"github.com/mandelsoft/kubecrtutils/objutils/objfilter"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/common"
	"github.com/mandelsoft/logging"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// HostedZoneReconciler reconciles a HostedZone object
type HostedZoneReconciler struct {
	*common.Reconciler
	Mode      Mode
	Finalizer string

	Manifests map[string][]byte

	Options     *Options
	Runtime     cluster.Cluster
	ownerFilter objfilter.Interface

	index index.UntypedIndex
}

func (r *HostedZoneReconciler) Request(def *reconciler.BaseRequest[*corednsv1alpha1.HostedZone]) reconciler.ReconcileRequest[*corednsv1alpha1.HostedZone] {
	return &ReconcileRequest{
		reconciler.DefaultReconcileRequest[*corednsv1alpha1.HostedZone, *HostedZoneReconciler]{
			BaseRequest: *def,
			Reconciler:  r,
		},
	}
}

func (r *HostedZoneReconciler) IsSeparateRuntime() bool {
	return r.Options.RuntimeNamespace != "" || !r.Dataplane.IsSameAs(r.Runtime)
}

func (r *HostedZoneReconciler) TriggerChildren(ctx context.Context, logger logging.Logger, obj client.ObjectKey) error {
	logger.Info("notify children about changes")
	children, err := r.GetNestedZones(ctx, obj.Namespace, obj.Name)
	if err != nil {
		return err
	}
	for _, c := range children {
		logger.Info("triggering child", "name", c.Name, "namespace", c.Namespace)
		r.Dataplane.EnqueueByObject(ctx, &c)
	}
	return nil
}

func (r *HostedZoneReconciler) TriggerEntries(ctx context.Context, logger logging.Logger, obj client.ObjectKey) error {
	entries, err := r.GetEntriesForZone(ctx, obj.Namespace, obj.Name)
	if err != nil {
		return err
	}
	logger.Info("notify {{amount}} children about changes", "amount", len(entries))
	for _, c := range entries {
		r.Dataplane.EnqueueByObject(ctx, &c)
	}
	return nil
}

func (r *HostedZoneReconciler) GetRootInfo(ctx *ReconcileRequest, obj *corednsv1alpha1.HostedZone) (*Responsibility, bool, Problem) {
	return common.GetRootInfo(ctx, ctx, ctx, obj)
}

func ConditionStatus(b bool) metav1.ConditionStatus {
	if b {
		return metav1.ConditionTrue
	}
	return metav1.ConditionFalse
}
