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

	"github.com/go-logr/logr"
	"github.com/mandelsoft/kubedns/pkg/enqueue"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
)

type Responsibility struct {
	Root    *corednsv1alpha1.HostedZone
	Runtime string
	Class   string
}

// HostedZoneReconciler reconciles a HostedZone object
type HostedZoneReconciler struct {
	Mode         Mode
	Finalizer    string
	FieldManager string
	DataPlaneURL string

	Manifests map[string][]byte

	Options   *Options
	Clientset *kubernetes.Clientset
	Mux       enqueue.Mux
	DataPlane client.Client
	Runtime   client.Client
	Scheme    *runtime.Scheme
}

// +kubebuilder:rbac:groups=core,resources=secrets;configmaps;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// +kubebuilder:rbac:groups=coredns.mandelsoft.org,resources=hostedzones,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coredns.mandelsoft.org,resources=hostedzones/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coredns.mandelsoft.org,resources=hostedzones/finalizers,verbs=update

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
	logger := logf.FromContext(ctx)

	logger.Info("called reconcile")
	// Fetch the object
	obj := &corednsv1alpha1.HostedZone{}
	if err := r.DataPlane.Get(ctx, req.NamespacedName, obj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	logger.Info("got object")

	// anonymous validation for responsibility fields
	if !IsASCIIAlnumString(String(obj.Spec.Runtime, "")) || !IsASCIIAlnumString(String(obj.Spec.Class, "")) {
		mod, err := r.UpdateCondition(ctx, obj, metav1.Condition{
			Type:               ValidationConditionType,
			Status:             metav1.ConditionFalse, // Use metav1 constant
			Reason:             ReasonInvalidParent,
			Message:            "class and runtime must use ASCII alphanumeric characters, only.",
			ObservedGeneration: obj.Generation,
		})
		if mod && err == nil {
			r.TriggerChildren(ctx, logger, req.NamespacedName)
		}
		return reconcile.Result{}, err
	}

	logger.Info("checking responsibility")
	// check responsibility
	ok, err := r.IsResponsibileFor(ctx, logger, obj)
	if !ok || err != nil {
		if !ok {
			logger.Info("not responsible for this zone")
		}
		return ctrl.Result{}, err
	}

	logger.Info("have to handle object")
	action := NewRequest(ctx, logger, r, obj)

	// Handle Deletion
	if !obj.ObjectMeta.DeletionTimestamp.IsZero() || r.ChangedResponsibility(ctx, obj) {
		if !obj.ObjectMeta.DeletionTimestamp.IsZero() {
			logger.Info("deletion of object requested")
		} else {
			logger.Info("responsibility changed", "parent", obj.Spec.ParentRef, "runtime", obj.Spec.Runtime, "class", obj.Spec.Class)
		}
		if controllerutil.ContainsFinalizer(obj, r.Finalizer) {
			logger.Info("already active -> delete dependent resources")
			// Run your external cleanup logic here
			if err := action.DeleteExternalResources(); err != nil {
				// If cleanup fails, we don't remove the finalizer;
				// we requeue to try again.
				return ctrl.Result{}, err
			}

			// Remove finalizer and update
			logger.Info("finally removing finalizer")
			controllerutil.RemoveFinalizer(obj, r.Finalizer)
			if err := r.DataPlane.Update(ctx, obj); err != nil {
				return ctrl.Result{}, err
			}
		}

		action.TriggerChildren()
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	// handle finalizer
	updated := false
	if obj.Spec.ParentRef != "" {
		// Remove finalizer from slave and update
		logger.Info("removing finalizer for slave zone")
		updated = controllerutil.RemoveFinalizer(obj, r.Finalizer)
		if obj.Status.Observed != nil {
			obj.Status.Observed = nil
			updated = true
		}

	} else {
		//  Add Finalizer for root zone
		updated = controllerutil.AddFinalizer(obj, r.Finalizer)
		if updated {
			logger.Info("taking responsibilty")
		}
	}
	if updated {
		if err := r.DataPlane.Update(ctx, obj); err != nil {
			return ctrl.Result{}, err
		}
	}

	//  Normal Business Logic
	logger.Info("Reconciling Hosted Zone", "name", obj.Name)
	return action.Reconcile()
}

func (r *HostedZoneReconciler) TriggerChildren(ctx context.Context, logger logr.Logger, obj client.ObjectKey) {
	logger.Info("notify children about changes")
	children := r.GetChildren(ctx, obj.Namespace, obj.Name)
	for _, c := range children {
		logger.Info("triggering child", "name", c.Name, "namespace", c.Namespace)
		r.Mux.EnqueueByObject(&c)
	}
}

func (r *HostedZoneReconciler) GetRootInfo(ctx context.Context, logger logr.Logger, obj *corednsv1alpha1.HostedZone) (*Responsibility, bool, error) {
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
		obj = &parent
	}
	return &Responsibility{Root: obj, Runtime: String(obj.Spec.Runtime, ""), Class: String(obj.Spec.Class, "")}, true, nil
}

func (r *HostedZoneReconciler) ChangedResponsibility(ctx context.Context, obj *corednsv1alpha1.HostedZone) bool {
	if !controllerutil.ContainsFinalizer(obj, r.Finalizer) {
		return false
	}
	if obj.Spec.ParentRef != "" {
		// zone has been moved and is no top-level zone anymore
		return true
	}
	if obj.Status.Observed == nil {
		// never handled
		return false
	}
	if obj.Status.Observed.Runtime != String(obj.Spec.Runtime, "") || obj.Status.Observed.Class != String(obj.Spec.Class, "") {
		// responsibility change has been requested
		return true
	}
	return false
}

func (r *HostedZoneReconciler) IsResponsibileFor(ctx context.Context, logger logr.Logger, obj *corednsv1alpha1.HostedZone) (bool, error) {
	if controllerutil.ContainsFinalizer(obj, r.Finalizer) {
		return true, nil
	}
	logger.Info("lookup root for {{name}}")
	info, ok, err := r.GetRootInfo(ctx, logger, obj)
	if err != nil {
		if ok {
			// dangling object, always report problem
			logger.Info("report dangling zone", "error", err.Error())
			_, err = r.UpdateCondition(ctx, obj, metav1.Condition{
				Type:               ValidationConditionType,
				Status:             metav1.ConditionFalse, // Use metav1 constant
				Reason:             ReasonInvalidParent,
				Message:            err.Error(),
				ObservedGeneration: obj.Generation,
			})
		}
		return false, err
	}
	new := info.Runtime == r.Options.Runtime && info.Class == r.Options.Class
	logger.Info("checking match", "root", info.Root.Name, "match", new, "runtime", info.Runtime, "class", info.Class)
	if info.Root.Status.Observed == nil {
		return new, nil
	}

	match := info.Root.Status.Observed.Class == r.Options.Class && info.Root.Status.Observed.Class == r.Options.Class
	logger.Info("checking registered match", "root", info.Root.Name, "match", new, "runtime", info.Root.Status.Observed.Runtime, "class", info.Root.Status.Observed.Class)
	return match, nil

}

func (r *HostedZoneReconciler) UpdateCondition(ctx context.Context, instance *corednsv1alpha1.HostedZone, c metav1.Condition, mod ...bool) (bool, error) {
	m := false
	for _, v := range mod {
		m = v || m
	}
	conditions := &instance.Status.Conditions

	if meta.SetStatusCondition(conditions, c) || m {
		if err := r.DataPlane.Status().Update(ctx, instance); err != nil {
			return false, fmt.Errorf("failed to update status after successful validation: %w", err)
		}
		return true, nil
	}
	return false, nil
}
