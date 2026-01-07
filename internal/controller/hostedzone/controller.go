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
	"fmt"
	"slices"
	"time"

	"github.com/mandelsoft/kubedns/internal/controller/common"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cluster"
	. "github.com/mandelsoft/kubedns/pkg/kubecrtutils/controller/controllerutils/reconcile"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/index"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/owner"
	"github.com/mandelsoft/logging"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
)

const INDEX_SASECFRET = "serviceaccount-secret"

type Responsibility struct {
	Root       *corednsv1alpha1.HostedZone
	Parent     *corednsv1alpha1.HostedZone
	RuntimeSet bool
	Runtime    string
	Class      string
}

// HostedZoneReconciler reconciles a HostedZone object
type HostedZoneReconciler struct {
	*common.Reconciler
	Mode         Mode
	Finalizer    string
	DataPlaneURL string

	Manifests map[string][]byte

	Options *Options
	Runtime cluster.Cluster

	runtimeOwner owner.OwnerHandler
	dns          DNSHandler
	recorder     record.EventRecorder

	index index.UntypedIndex
}

func (r *HostedZoneReconciler) IsSeparateRuntime() bool {
	return r.Options.RuntimeNamespace != "" || !r.DataPlane.IsSameAs(r.Runtime)
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the HostedZone object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.4/pkg/reconcile
func (r *HostedZoneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.WithName(req.String()).WithValues("object", req.NamespacedName)

	var after time.Duration

	// Fetch the object
	obj := &corednsv1alpha1.HostedZone{}
	if err := r.DataPlane.Get(ctx, req.NamespacedName, obj); err != nil {
		if !errors.IsNotFound(err) {
			logger.Info("error getting object to reconcile", "error", err)
			return ctrl.Result{}, err
		}
		logger.Info("Deleted hostedzone")
		r.TriggerEntries(ctx, logger, req.NamespacedName)
		obj = nil
	} else {
		if obj.DeletionTimestamp.IsZero() {
			logger.Info("Reconcile hostedzone")
			after = 300 * time.Second
		} else {
			logger.Info("Delete hostedzone")
		}
	}

	if obj != nil {
		// anonymous validation for responsibility fields
		if !IsASCIIAlnumString(String(obj.Spec.Runtime, "")) || !IsASCIIAlnumString(String(obj.Spec.Class, "")) {
			mod, err := r.UpdateCondition(ctx, logger, obj, metav1.Condition{
				Type:               corednsv1alpha1.ValidationConditionType,
				Status:             metav1.ConditionFalse, // Use metav1 constant
				Reason:             corednsv1alpha1.ReasonInvalidParent,
				Message:            "class and runtime must use ASCII alphanumeric characters, only.",
				ObservedGeneration: obj.Generation,
			})
			if mod && err == nil {
				r.TriggerChildren(ctx, logger, req.NamespacedName)
				r.TriggerEntries(ctx, logger, req.NamespacedName)
			}
			return reconcile.Result{}, err
		}
	}

	action := NewRequest(ctx, logger, r, req.NamespacedName, obj)
	prob := action.Reconcile()
	prob = AggregateProblem(prob, TemporaryProblem(action.Update()))

	return Result(logger, prob, after)
}

func (r *HostedZoneReconciler) TriggerChildren(ctx context.Context, logger logging.Logger, obj client.ObjectKey) error {
	logger.Info("notify children about changes")
	children, err := r.GetNestedZones(ctx, obj.Namespace, obj.Name)
	if err != nil {
		return err
	}
	for _, c := range children {
		logger.Info("triggering child", "name", c.Name, "namespace", c.Namespace)
		r.DataPlane.EnqueueByObject(&c)
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
		r.DataPlane.EnqueueByObject(&c)
	}
	return nil
}

func (r *HostedZoneReconciler) GetRootInfo(ctx context.Context, logger logging.Logger, obj *corednsv1alpha1.HostedZone) (*Responsibility, bool, Problem) {
	var path string
	var directParent *corednsv1alpha1.HostedZone

	hist := []string{obj.GetName()}
	for obj.Spec.ParentRef != "" {
		var parent corednsv1alpha1.HostedZone
		path = path + "/" + obj.Spec.ParentRef
		logger.Info("handle parent", "parent", path)
		if slices.Contains(hist, obj.Spec.ParentRef) {
			return nil, true, Failedf("reference cyle %s", path)
		}
		err := r.DataPlane.Get(ctx, client.ObjectKey{obj.GetNamespace(), obj.Spec.ParentRef}, &parent)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil, true, Failedf("parent %s not found", obj.Spec.ParentRef)
			}
			return nil, false, TemporaryProblem(err)
		}
		if !parent.GetDeletionTimestamp().IsZero() {
			return nil, true, Failedf("parent %s deleted", obj.Spec.ParentRef)
		}
		if directParent == nil {
			directParent = &parent
		}
		obj = &parent
	}
	return &Responsibility{Root: obj, Parent: directParent, Runtime: String(obj.Spec.Runtime, ""), RuntimeSet: obj.Spec.Runtime != nil, Class: String(obj.Spec.Class, "")}, true, nil
}

func (r *HostedZoneReconciler) UpdateCondition(ctx context.Context, logger logging.Logger, instance *corednsv1alpha1.HostedZone, c metav1.Condition, mod ...bool) (bool, error) {
	conditions := &instance.Status.Conditions
	m := meta.SetStatusCondition(conditions, c)
	if m {
		r.recorder.Eventf(instance, corev1.EventTypeNormal, c.Type, c.Message)
	}

	for _, v := range mod {
		m = v || m
	}

	if m {
		logger.Info("status needs update")
		if err := r.DataPlane.Status().Update(ctx, instance); err != nil {
			return false, fmt.Errorf("failed to update status after successful validation: %w", err)
		}
		return true, nil
	}
	return false, nil
}

func ConditionStatus(b bool) metav1.ConditionStatus {
	if b {
		return metav1.ConditionTrue
	}
	return metav1.ConditionFalse
}
