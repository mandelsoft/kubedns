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
	errors2 "errors"
	"fmt"

	"github.com/mandelsoft/kubedns/pkg/clusterutils"
	"github.com/mandelsoft/kubedns/pkg/enqueue"
	"github.com/mandelsoft/kubedns/pkg/index"
	"github.com/mandelsoft/kubedns/pkg/owner"
	"github.com/mandelsoft/logging"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
)

const INDEX_SASECFRET = "serviceaccount-secret"

type Responsibility struct {
	Root    *corednsv1alpha1.HostedZone
	Runtime string
	Class   string
}

// HostedZoneReconciler reconciles a HostedZone object
type HostedZoneReconciler struct {
	logging.Logger
	Mode         Mode
	Finalizer    string
	FieldManager string
	DataPlaneURL string

	Manifests map[string][]byte

	Options      *Options
	Mux          enqueue.Mux
	DataPlane    clusterutils.Cluster
	Runtime      clusterutils.Cluster
	runtimeOwner owner.OwnerHandler
	dns          DNSHandler

	index index.UntypedIndex
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
	logger := Log.WithName(req.NamespacedName.String()).WithValues("name", req.NamespacedName)

	logger.Info("reconciling")
	// Fetch the object
	obj := &corednsv1alpha1.HostedZone{}
	if err := r.DataPlane.Get(ctx, req.NamespacedName, obj); err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		logger.Info("hosted zone object deleted")
	} else {
		logger.Info("hosted zone object found")
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
			}
			return reconcile.Result{}, err
		}
	}

	action := NewRequest(ctx, logger, r, req.NamespacedName, obj)
	err := action.Reconcile()
	err = errors2.Join(err, action.Update())
	return ctrl.Result{}, err
}

func (r *HostedZoneReconciler) TriggerChildren(ctx context.Context, logger logging.Logger, obj client.ObjectKey) {
	logger.Info("notify children about changes")
	children := r.GetChildren(ctx, obj.Namespace, obj.Name)
	for _, c := range children {
		logger.Info("triggering child", "name", c.Name, "namespace", c.Namespace)
		r.Mux.EnqueueByObject(&c)
	}
}

func (r *HostedZoneReconciler) GetRootInfo(ctx context.Context, logger logging.Logger, obj *corednsv1alpha1.HostedZone) (*Responsibility, bool, error) {
	var parent corednsv1alpha1.HostedZone
	var path string

	for obj.Spec.ParentRef != "" {
		path = path + "/" + obj.Spec.ParentRef
		logger.Info("handle parent", "parent", path)
		err := r.DataPlane.Get(ctx, client.ObjectKey{obj.GetNamespace(), obj.Spec.ParentRef}, &parent)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil, true, fmt.Errorf("parent %s not found", obj.Spec.ParentRef)
			}
			return nil, false, err
		}
		if !parent.GetDeletionTimestamp().IsZero() {
			return nil, true, fmt.Errorf("parent %s deleted", obj.Spec.ParentRef)
		}
		obj = &parent
	}
	return &Responsibility{Root: obj, Runtime: String(obj.Spec.Runtime, ""), Class: String(obj.Spec.Class, "")}, true, nil
}

func (r *HostedZoneReconciler) UpdateCondition(ctx context.Context, logger logging.Logger, instance *corednsv1alpha1.HostedZone, c metav1.Condition, mod ...bool) (bool, error) {
	m := false
	for _, v := range mod {
		m = v || m
	}

	conditions := &instance.Status.Conditions

	if meta.SetStatusCondition(conditions, c) || m {
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
