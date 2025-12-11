package assure

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ApplyManifestUnstructured creates or updates a resource defined by the manifestBytes.
// owner must be the object that manages this resource (for garbage collection).
// scheme is needed to map GVK to the owner.
func ApplyManifestUnstructured(ctx context.Context, k8sClient client.Client, scheme *runtime.Scheme, owner client.Object, manifestBytes []byte) error {

	// --- Decode Manifest into Unstructured Object ---

	// Use YAMLSerializer for decoding YAML/JSON into a generic runtime.Object.
	// We use the UnstructuredSerializer because we don't know the exact Go struct type yet.
	yamlSerializer := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

	// Create an empty Unstructured object to receive the decoded data.
	obj := &unstructured.Unstructured{}

	// Decode the manifest bytes into the Unstructured object.
	_, _, err := yamlSerializer.Decode(manifestBytes, nil, obj)
	if err != nil {
		return fmt.Errorf("failed to decode manifest into Unstructured: %w", err)
	}

	// Set the namespace of the created object to match the owner's namespace
	obj.SetNamespace(owner.GetNamespace())

	// ---  Check for Existence and Apply Patch ---

	// We need the latest version of the object to perform the patch or creation check.
	currentObj := obj.DeepCopy() // Copy the GVK, Name, and Namespace for the Get call

	// Fetch the current state from the cluster
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(currentObj), currentObj); err != nil {
		// If the resource is not found, we will create it.
		if client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("failed to get current object state: %w", err)
		}
	} else {
		// Resource exists: Update the desired object's ResourceVersion for potential SSA reuse
		// This is not strictly required for SSA but helps in logging/tracing.
		obj.SetResourceVersion(currentObj.GetResourceVersion())
	}

	// --- Set Owner Reference for Garbage Collection ---

	// Set the owner reference on the desired object before applying.
	// This ensures the resource is deleted when the owning custom resource is deleted.
	if err := controllerutil.SetOwnerReference(owner, obj, scheme); err != nil {
		return fmt.Errorf("failed to set owner reference: %w", err)
	}

	// Apply the Object using Server-Side Apply (SSA) ---

	// Ensure the object has the necessary TypeMeta
	if obj.GetKind() == "" || obj.GetAPIVersion() == "" {
		// Fallback/check if decoding failed to populate GVK
		gvk := currentObj.GetObjectKind().GroupVersionKind()
		obj.SetKind(gvk.Kind)
		obj.SetAPIVersion(gvk.GroupVersion().String())
	}

	if annos := currentObj.GetAnnotations(); annos != nil {
		for k, v := range obj.GetAnnotations() {
			annos[k] = v
		}
		obj.SetAnnotations(annos)
	}

	if labels := currentObj.GetLabels(); labels != nil {
		for k, v := range obj.GetLabels() {
			labels[k] = v
		}
		obj.SetLabels(labels)
	}

	// Clean up the .status field (Always managed by the server/controller, not the manifest)
	delete(obj.Object, "status")

	// 4. Calculate the patch. This compares originalForPatch (live) against desiredObj (intended).
	patch := client.MergeFrom(currentObj)
	patchData, err := patch.Data(obj)
	if err != nil {
		return fmt.Errorf("failed to calculate patch data: %w", err)
	}

	// heck if the patch is effectively empty.
	patchStr := string(patchData)
	if patchStr == "{}" || patchStr == "[]" {
		return nil // No change needed. Skip API call.
	}

	// 6. Apply Patch (API call is sent here).
	if err := k8sClient.Patch(ctx, obj, patch); err != nil {
		return fmt.Errorf("failed to patch object: %w", err)
	}
	return nil
}

const (
	// Base domain for cross-cluster labels/annotations
	OwnerLabelDomain = "ownership.my-multi-cluster.io"
)

// SetGeneralizedOwnerReference sets the owner relationship either locally (in-cluster)
// or via annotations (cross-cluster), based on the sourceClusterID.
//
// Arguments:
//
//	dependent: The resource being managed (the Pod, Deployment, etc.).
//	owner: The primary resource (e.g., your Custom Resource instance).
//	scheme: The runtime.Scheme for GVK lookup.
//	sourceClusterID: If non-empty, sets cross-cluster annotations instead of OwnerReference.
func SetGeneralizedOwnerReference(
	dependent client.Object,
	owner client.Object,
	scheme *runtime.Scheme,
	sourceClusterID string,
) error {

	if sourceClusterID == "" {
		// --- 1. IN-CLUSTER MODE (Standard Kubernetes Garbage Collection) ---

		// Use the controller-runtime helper to set the OwnerReference in metadata.
		return controllerutil.SetOwnerReference(owner, dependent, scheme)

	} else {
		// --- 2. CROSS-CLUSTER MODE (Label-Based Ownership) ---

		// Get the GVK of the owner resource
		ownerGVK, err := apiutil.GVKForObject(owner, scheme)
		if err != nil {
			return fmt.Errorf("failed to get GVK for owner: %w", err)
		}

		info := dependent.GetLabels()
		if info == nil {
			info = make(map[string]string)
		}

		// Set the necessary annotations to identify the remote owner.
		info[fmt.Sprintf("%s/owner-gvk", OwnerLabelDomain)] = ownerGVK.String()
		info[fmt.Sprintf("%s/owner-namespace", OwnerLabelDomain)] = owner.GetNamespace()
		info[fmt.Sprintf("%s/owner-name", OwnerLabelDomain)] = owner.GetName()
		info[fmt.Sprintf("%s/source-cluster-id", OwnerLabelDomain)] = sourceClusterID

		dependent.SetLabels(info)

		// CRITICAL: Ensure no native OwnerReference is set,
		// otherwise the current cluster's GC might try to delete the resource when the local owner disappears.
		dependent.SetOwnerReferences(nil)

		return nil
	}
}
