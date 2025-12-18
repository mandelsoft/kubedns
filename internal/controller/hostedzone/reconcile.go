package hostedzone

import (
	"context"
	"fmt"
	"reflect"

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
	logr.Logger
	instance   *corev1alpha1.HostedZone
	reconciler *HostedZoneReconciler
	ctx        context.Context
}

func NewRequest(ctx context.Context, l logr.Logger, reconciler *HostedZoneReconciler, instance *corev1alpha1.HostedZone) *ReconcileRequest {
	return &ReconcileRequest{
		Logger:     l,
		ctx:        ctx,
		reconciler: reconciler,
		instance:   instance,
	}
}

func (r *ReconcileRequest) Reconcile() (ctrl.Result, error) {
	reason, err := r.Validate()
	if err != nil {
		if reason != "" {
			// --- Validation Failed: Use meta.SetStatusCondition to set ConditionFalse ---
			if err = r.reconciler.UpdateCondition(r.ctx, r.instance, metav1.Condition{
				Type:               ValidationConditionType,
				Status:             metav1.ConditionFalse, // Use metav1 constant
				Reason:             reason,
				Message:            err.Error(),
				ObservedGeneration: r.instance.Generation,
			}); err != nil {
				return reconcile.Result{}, err
			}
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
	if err = r.reconciler.UpdateCondition(r.ctx,
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

	return r.HandleExternalResources()

}
func (r *ReconcileRequest) HandleExternalResources() (ctrl.Result, error) {
	return ctrl.Result{}, nil

	if r.reconciler.IsSeparateRuntime() {
		// assure target namespace
		var ns v1.Namespace
		namespace := r.reconciler.Mode.GetRuntimeNamespace(r)
		err := r.reconciler.Runtime.Get(r.ctx, client.ObjectKey{Name: namespace}, &ns)
		if err != nil {
			if errors.IsNotFound(err) {
				r.Info("assure target namespace", "namespace", namespace)
				ns.Name = r.reconciler.Mode.GetRuntimeNamespace(r)
				err = r.reconciler.Runtime.Create(r.ctx, &ns)
			}
		}
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	values := r.Values()

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

	access := r.reconciler.Mode.AccessValues(r)

	if !r.reconciler.IsSeparateRuntime() || access != nil {
		if !reflect.DeepEqual(access, GetAccessValues(values)) {
			values["dataplane"] = merge(values["dataplane"].(map[string]interface{}), access)
			_, runtime, err = render.Render(r.reconciler.Manifests, values)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		for _, data := range runtime {
			_, err := r.ClientSideApply("runtime", r.reconciler.Runtime, data)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	return ctrl.Result{}, nil
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
	err = clt.Patch(r.ctx, obj, client.Apply, &client.PatchOptions{
		FieldManager: r.reconciler.FieldManager,
		Force:        generics.PointerTo(true),
	})

	if err != nil {
		return fmt.Errorf("failed to apply manifest: %w", err)
	}

	return nil
}

func (r *ReconcileRequest) ClientSideApply(target string, clt client.Client, manifest []byte) (*unstructured.Unstructured, error) {
	// 1. Decode bytes into a 'desired' unstructured object
	desired := &unstructured.Unstructured{}
	dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	_, _, err := dec.Decode(manifest, nil, desired)
	if err != nil {
		return nil, err
	}

	// 2. Try to get the current object from the cluster
	current := &unstructured.Unstructured{}
	current.SetGroupVersionKind(desired.GroupVersionKind())
	err = clt.Get(r.ctx, client.ObjectKey{
		Namespace: desired.GetNamespace(),
		Name:      desired.GetName(),
	}, current)

	if errors.IsNotFound(err) {
		r.Info("creating resource", "target", target, "name", desired.GetName(), "namespace", desired.GetNamespace(), "groupkind", desired.GroupVersionKind())
		// PATH A: Create if not found
		return desired, clt.Create(r.ctx, desired, &client.CreateOptions{
			FieldManager: r.reconciler.FieldManager,
		})
	} else if err != nil {
		return desired, err
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
	patchData, err := patch.Data(current)
	if err != nil {
		return nil, err
	}

	rawPatch := client.RawPatch(types.MergePatchType, patchData)
	if string(patchData) == "{}" {
		r.Info("resource uptodate", "target", target, "name", desired.GetName(), "namespace", desired.GetNamespace(), "groupkind", desired.GroupVersionKind())

		return desired, nil // No changes, exit early
	}

	r.Info("apply patch", "target", target, "name", desired.GetName(), "namespace", desired.GetNamespace(), "groupkind", desired.GroupVersionKind())
	return desired, clt.Patch(r.ctx, current, rawPatch, &client.PatchOptions{
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

func (r *ReconcileRequest) DeploymentName(instance *corev1alpha1.HostedZone) string {
	return fmt.Sprintf("dns-%s-%s", instance.Namespace, instance.Name)
}

func (r *ReconcileRequest) Values() map[string]interface{} {
	name := fmt.Sprintf("%s-%s", r.instance.Namespace, r.instance.Name)
	return map[string]interface{}{
		"runtime": map[string]interface{}{
			"namespace": r.reconciler.Mode.GetRuntimeNamespace(r),
			"separated": r.reconciler.IsSeparateRuntime(),
		},
		"dataplane": merge(map[string]interface{}{
			"namespace": r.instance.Namespace,
			"server":    r.reconciler.DataPlaneURL,
			"token":     "",
			"cadata":    "",
		}, r.reconciler.Mode.AccessValues(r)),
		"deployment": map[string]interface{}{
			"name":     r.DeploymentName(r.instance),
			"label":    "dns-service-" + name,
			"replicas": 1,
		},
		"service": map[string]interface{}{
			"name": "dns-server-" + name,
		},
		"config": map[string]interface{}{
			"name": "dns-server-" + name,
			"zone": r.instance.Name,
		},
	}

}

func merge(dst, src map[string]interface{}) map[string]interface{} {
	if dst == nil {
		dst = make(map[string]interface{})
	}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
