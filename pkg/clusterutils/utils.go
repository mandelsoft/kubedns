package clusterutils

import (
	"context"
	"fmt"

	"github.com/mandelsoft/goutils/generics"
	"github.com/mandelsoft/kubedns/pkg/merge"
	"github.com/mandelsoft/kubedns/pkg/objutils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type OperationContext interface {
	context.Context
	Logger
	GetFieldManager() string
	Modify(obj client.Object) error
}

type ObjectModifier func(obj client.Object) error

type _defaultOprationContext struct {
	context.Context
	Logger
	fieldManager string
	modifer      ObjectModifier
}

func aggregatedModifier(mod ...ObjectModifier) ObjectModifier {
	if len(mod) == 1 {
		return mod[0]
	}
	// assure non-nil modifier
	return func(obj client.Object) error {
		for _, m := range mod {
			err := m(obj)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func DefaultOperationContext(ctx context.Context, logger Logger, fm string, m ...ObjectModifier) OperationContext {
	return &_defaultOprationContext{
		Context:      ctx,
		Logger:       logger,
		fieldManager: fm,
		modifer:      aggregatedModifier(m...),
	}
}

func (o *_defaultOprationContext) GetFieldManager() string {
	return o.fieldManager
}

func (o *_defaultOprationContext) Modify(obj client.Object) error {
	return o.modifer(obj)
}

type Logger interface {
	Info(msg string, args ...interface{})
}

func DeleteObject(c Cluster, ctx OperationContext, manifest []byte) error {
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
	err = c.Get(ctx, key, &current)

	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if !current.GetDeletionTimestamp().IsZero() {
		return fmt.Errorf("resource %q is being deleted", key)
	}
	ctx.Info("deleting object", "name", key.Name, "namespace", key.Namespace, "cluster", c.GetName(), "groupkind", obj.GroupVersionKind())
	err = c.Delete(ctx, &obj)
	return err
}

// ServerSideApply ensures the cluster state matches the manifest without
// wiping out system-generated data or status.
func ServerSideApply(c Cluster, ctx OperationContext, manifest []byte) error {
	// 1. Decode the raw bytes into an Unstructured object.
	// We use Unstructured to avoid needing the Go types for every manifest.
	obj := &unstructured.Unstructured{}
	dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	_, _, err := dec.Decode(manifest, nil, obj)
	if err != nil {
		return fmt.Errorf("failed to decode manifest: %w", err)
	}

	err = ctx.Modify(obj)
	if err != nil {
		return fmt.Errorf("failed to modify manifest: %w", err)
	}

	// 2. Perform Server-Side Apply.
	// - client.Apply: Tells K8s to merge this with the existing object.
	// - FieldManager: Identifies your controller as the owner of THESE specific fields.
	// - ForceOwnership: If a human manually changed a field you own, this overrides it.
	err = c.Patch(ctx, obj, client.Apply, &client.PatchOptions{
		FieldManager: ctx.GetFieldManager(),
		Force:        generics.PointerTo(true),
	})

	if err != nil {
		return fmt.Errorf("failed to apply manifest: %w", err)
	}

	return nil
}

func ClientSideApply(c Cluster, ctx OperationContext, manifest []byte) (*unstructured.Unstructured, error) {
	// 1. Decode bytes into a 'desired' unstructured object
	desired := unstructured.Unstructured{}
	dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	_, _, err := dec.Decode(manifest, nil, &desired)
	if err != nil {
		return nil, err
	}

	// 2. give context the chance to modify object
	err = ctx.Modify(&desired)
	if err != nil {
		return nil, fmt.Errorf("failed to modify manifest: %w", err)
	}

	current := unstructured.Unstructured{}
	current.SetGroupVersionKind(desired.GroupVersionKind())
	err = c.Get(ctx, client.ObjectKey{
		// Try to get the current object from the cluster
		Namespace: desired.GetNamespace(),
		Name:      desired.GetName(),
	}, &current)

	if errors.IsNotFound(err) {
		ctx.Info("creating resource", "cluster", c.GetName(), "name", desired.GetName(), "namespace", desired.GetNamespace(), "groupkind", desired.GroupVersionKind())
		return &desired, c.Create(ctx, &desired, &client.CreateOptions{
			// PATH A: Create if not found
			FieldManager: ctx.GetFieldManager(),
		})
	} else if err != nil {
		return &desired, err
	}

	m, err := merge.NewObjectMerger(c.GetTypeConverter(), ctx.GetFieldManager())
	if err != nil {
		return nil, err
	}

	tmp, err := m.MergeObservingManagedFields(&current, &desired)
	if err != nil {
		return nil, err
	}

	// PATH B: Patch existing object
	// We use 'current' as the base. We only want to update the 'spec' (or other non-system fields).
	// IMPORTANT: To preserve status/finalizers, we ensure they aren't overwritten in 'desired'.

	// Create a patch object that calculates the diff between 'current' and 'desired'
	patch := client.MergeFrom(current.DeepCopy())

	// Apply the patch to 'current' using our 'desired' state
	// Note: We update 'current' with 'desired' fields here

	for k, v := range tmp.Object {
		if k != "metadata" {
			current.Object[k] = v
		}
	}
	for k, v := range desired.GetAnnotations() {
		objutils.SetAnnotation(&current, k, v)
	}
	current.SetLabels(desired.GetLabels())

	patchData, err := patch.Data(&current)
	if err != nil {
		return nil, err
	}

	rawPatch := client.RawPatch(types.MergePatchType, patchData)
	if string(patchData) == "{}" {
		ctx.Info("resource uptodate", "cluster", c.GetName(), "name", desired.GetName(), "namespace", desired.GetNamespace(), "groupkind", desired.GroupVersionKind())

		return &desired, nil // No changes, exit early
	}

	ctx.Info("apply patch", "cluster", c.GetName(), "name", desired.GetName(), "namespace", desired.GetNamespace(), "groupkind", desired.GroupVersionKind(), "patch", string(patchData))
	return &desired, c.Patch(ctx, &current, rawPatch, &client.PatchOptions{
		FieldManager: ctx.GetFieldManager(),
	})
}

func GKVForObject(c SchemeProvider, obj client.Object) (schema.GroupVersionKind, error) {
	return apiutil.GVKForObject(obj, c.GetScheme())
}

func GKForObject(c SchemeProvider, obj client.Object) (schema.GroupKind, error) {
	gkv, err := apiutil.GVKForObject(obj, c.GetScheme())
	if err != nil {
		return schema.GroupKind{}, err
	}
	return schema.GroupKind{Group: gkv.Group, Kind: gkv.Kind}, nil
}

type modificationWrapper struct {
	OperationContext
	mod ObjectModifier
}

func WithModification(ctx OperationContext, mod ...ObjectModifier) OperationContext {
	return &modificationWrapper{
		OperationContext: ctx,
		mod:              aggregatedModifier(mod...),
	}
}

func (w *modificationWrapper) Modify(obj client.Object) error {
	err := w.OperationContext.Modify(obj)
	if err != nil {
		return err
	}
	return w.mod(obj)
}
