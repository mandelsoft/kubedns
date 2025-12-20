package hostedzone

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/goutils/generics"
	corev1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/pkg/render"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type ReconcileRequest struct {
	ReconcileContext
	reconciler *HostedZoneReconciler
	instance   *corev1alpha1.HostedZone
}

func NewRequest(ctx context.Context, l logr.Logger, reconciler *HostedZoneReconciler, instance *corev1alpha1.HostedZone) *ReconcileRequest {
	return &ReconcileRequest{
		ReconcileContext: NewReconcileContext(ctx, l, reconciler.DataPlaneURL, client.ObjectKeyFromObject(instance)),
		reconciler:       reconciler,
		instance:         instance,
	}
}

func (r *ReconcileRequest) Reconcile() (ctrl.Result, error) {
	mod := false

	reason, err := r.Validate()
	if err != nil {
		if reason != "" {
			// --- Validation Failed: Use meta.SetStatusCondition to set ConditionFalse ---
			if mod, err = r.reconciler.UpdateCondition(r, r.instance, metav1.Condition{
				Type:               ValidationConditionType,
				Status:             metav1.ConditionFalse, // Use metav1 constant
				Reason:             reason,
				Message:            err.Error(),
				ObservedGeneration: r.instance.Generation,
			}); err != nil {
				return reconcile.Result{}, err
			}
		}
		if mod {
			r.TriggerChildren()
		}
		return reconcile.Result{}, err
	}

	var observed *corev1alpha1.Observed
	if r.instance.Spec.ParentRef == "" {
		observed = r.instance.Status.Observed
		if observed == nil {
			r.Logger.Info("registering responsibility")
			observed = &corev1alpha1.Observed{
				Runtime: r.reconciler.Options.Runtime,
				Class:   r.reconciler.Options.Class,
			}
		}
	}

	r.Logger.Info("validation succeeded")
	// --- Validation Succeeded: Use meta.SetStatusCondition to set ConditionTrue ---
	if mod, err = r.reconciler.UpdateCondition(r,
		r.instance, metav1.Condition{
			Type:    ValidationConditionType,
			Status:  metav1.ConditionTrue,
			Reason:  reason,
			Message: "The HostedZone specification passed all validation checks.",
		},
		SetPointerField(&r.instance.Status.Observed, observed),
	); err != nil {
		return reconcile.Result{}, err
	}

	if mod {
		r.TriggerChildren()
	}

	if r.instance.Spec.ParentRef == "" {
		return reconcile.Result{}, nil
	}
	return r.HandleExternalResources()

}
func (r *ReconcileRequest) HandleExternalResources() (ctrl.Result, error) {
	return ctrl.Result{}, nil

	key := client.ObjectKeyFromObject(r.instance)

	if r.reconciler.IsSeparateRuntime() {
		// assure target namespace
		var ns v1.Namespace
		namespace := r.reconciler.Mode.RuntimeNamespace(key)
		err := r.reconciler.Runtime.Get(r, client.ObjectKey{Name: namespace}, &ns)
		if err != nil {
			if errors.IsNotFound(err) {
				r.Info("assure target namespace", "namespace", namespace)
				ns.Name = r.reconciler.Mode.RuntimeNamespace(key)
				err = r.reconciler.Runtime.Create(r, &ns)
			}
		}
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	values, repeat := r.Values(r.reconciler.Mode)

	// even if access for credentials has been failed, the dataplane is applied.
	// this may create a secret required to get the access token.

	r.Logger.Info("updating dataplane")
	dataplane, runtime, err := render.Render(r.reconciler.Manifests, values)
	if err != nil {
		return ctrl.Result{}, err
	}

	for _, data := range dataplane {
		_, err := r.ClientSideApply("dataplane", r.reconciler.DataPlane, data)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// if credential access failed, retry the reconcilation
	if repeat != nil {
		return ctrl.Result{}, repeat
	}

	if false {
		r.Logger.Info("updating runtime")
		for _, data := range runtime {
			_, err := r.ClientSideApply("runtime", r.reconciler.Runtime, data)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		r.Logger.Info("skipping runtime deployment")
	}

	return ctrl.Result{}, nil
}

func (r *ReconcileRequest) TriggerChildren() {
	r.reconciler.TriggerChildren(r, r.Logger, client.ObjectKeyFromObject(r.instance))
}

// ServerSideApply ensures the cluster state matches the manifest without
// wiping out system-generated data or status.
func (r *ReconcileRequest) ServerSideApply(target string, clt client.Client, manifest []byte) error {
	// 1. Decode the raw bytes into an Unstructured object.
	// We use Unstructured to avoid needing the Go types for every manifest.
	obj := &unstructured.Unstructured{}
	dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	_, _, err := dec.Decode(manifest, nil, obj)
	if err != nil {
		return fmt.Errorf("failed to decode manifest: %w", err)
	}

	// 2. Perform Server-Side Apply.
	// - client.Apply: Tells K8s to merge this with the existing object.
	// - FieldManager: Identifies your controller as the owner of THESE specific fields.
	// - ForceOwnership: If a human manually changed a field you own, this overrides it.
	err = clt.Patch(r, obj, client.Apply, &client.PatchOptions{
		FieldManager: r.reconciler.FieldManager,
		Force:        generics.PointerTo(true),
	})

	if err != nil {
		return fmt.Errorf("failed to apply manifest: %w", err)
	}

	return nil
}

func (r *ReconcileRequest) Delete(target string, clt client.Client, manifest []byte) error {
	obj := unstructured.Unstructured{}
	dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	_, _, err := dec.Decode(manifest, nil, &obj)
	if err != nil {
		return err
	}

	current := unstructured.Unstructured{}
	current.SetGroupVersionKind(obj.GroupVersionKind())
	key := client.ObjectKey{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
	err = clt.Get(r, key, &current)

	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if !current.GetDeletionTimestamp().IsZero() {
		return fmt.Errorf("resource %q is still being deleted", key)
	}
	r.Info("deleting object", "name", key.Name, "namespace", key.Namespace, "target", target, "groupkind", obj.GroupVersionKind())
	err = clt.Delete(r, &obj)
	return err
}

func (r *ReconcileRequest) ClientSideApply(target string, clt client.Client, manifest []byte) (*unstructured.Unstructured, error) {
	// 1. Decode bytes into a 'desired' unstructured object
	desired := unstructured.Unstructured{}
	dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	_, _, err := dec.Decode(manifest, nil, &desired)
	if err != nil {
		return nil, err
	}

	// 2. Try to get the current object from the cluster
	current := unstructured.Unstructured{}
	current.SetGroupVersionKind(desired.GroupVersionKind())
	err = clt.Get(r, client.ObjectKey{
		Namespace: desired.GetNamespace(),
		Name:      desired.GetName(),
	}, &current)

	if errors.IsNotFound(err) {
		r.Info("creating resource", "target", target, "name", desired.GetName(), "namespace", desired.GetNamespace(), "groupkind", desired.GroupVersionKind())
		// PATH A: Create if not found
		return &desired, clt.Create(r, &desired, &client.CreateOptions{
			FieldManager: r.reconciler.FieldManager,
		})
	} else if err != nil {
		return &desired, err
	}

	// PATH B: Patch existing object
	// We use 'current' as the base. We only want to update the 'spec' (or other non-system fields).
	// IMPORTANT: To preserve status/finalizers, we ensure they aren't overwritten in 'desired'.

	// Create a patch object that calculates the diff between 'current' and 'desired'
	patch := client.MergeFrom(current.DeepCopy())

	// Apply the patch to 'current' using our 'desired' state
	// Note: We update 'current' with 'desired' fields here
	current.Object["spec"] = desired.Object["spec"]
	current.SetLabels(desired.GetLabels())
	current.SetAnnotations(desired.GetAnnotations())
	patchData, err := patch.Data(&current)
	if err != nil {
		return nil, err
	}

	rawPatch := client.RawPatch(types.MergePatchType, patchData)
	if string(patchData) == "{}" {
		r.Info("resource uptodate", "target", target, "name", desired.GetName(), "namespace", desired.GetNamespace(), "groupkind", desired.GroupVersionKind())

		return &desired, nil // No changes, exit early
	}

	r.Info("apply patch", "target", target, "name", desired.GetName(), "namespace", desired.GetNamespace(), "groupkind", desired.GroupVersionKind())
	return &desired, clt.Patch(r, &current, rawPatch, &client.PatchOptions{
		FieldManager: r.reconciler.FieldManager,
	})
}

func GetAccessValues(in map[string]interface{}) map[string]interface{} {
	d := in["dataplane"].(map[string]interface{})

	t := d["token"]
	c := d["cdata"]
	if t != nil && c != nil {
		return map[string]interface{}{
			"token": t,
			"cdata": c,
		}
	}
	return nil
}
