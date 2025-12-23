package merge

import (
	"bytes"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	// "k8s.io/apimachinery/pkg/util/managedfields"
	"sigs.k8s.io/structured-merge-diff/v6/fieldpath"
)

func (m *ObjectMerger) MergeObservingManagedFields(liveObj, desiredObj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	// converter := managedfields.NewDeducedTypeConverter()
	converter := m.converter

	// 1. Convert to Typed Values
	liveTyped, err := converter.ObjectToTyped(liveObj)
	if err != nil {
		return nil, err
	}
	desiredTyped, err := converter.ObjectToTyped(desiredObj)
	if err != nil {
		return nil, err
	}

	// 2. Calculate the "Intent" (What are we trying to change?)
	comparison, _ := liveTyped.Compare(desiredTyped)
	deltaFields := comparison.Modified.Union(comparison.Added)

	// 3. Aggregate fields owned by OTHERS
	othersManagedFields := &fieldpath.Set{}
	for _, entry := range liveObj.GetManagedFields() {
		if entry.Manager == m.managerName {
			continue
		}
		otherFs := &fieldpath.Set{}
		if err := otherFs.FromJSON(bytes.NewReader(entry.FieldsV1.Raw)); err == nil {
			othersManagedFields = othersManagedFields.Union(otherFs)
		}
	}

	// 4. Calculate Conflicts (Manual Intersection)
	// We want to keep only the parts of our delta that are NOT in othersManagedFields.
	// Logic: Fields we want to touch MINUS fields others own = Safe fields to apply.
	safeToApply := deltaFields.Difference(othersManagedFields)

	// 5. Construct the final intent
	// We start with the Live object and only apply the 'safeToApply'
	// fields from our desired state.

	// We use recursive filtering to ensure 'desiredTyped' only contains
	// paths present in 'safeToApply'.
	filteredDesired := desiredTyped.ExtractItems(safeToApply)

	// 6. Final Merge
	// Result = Live State + Our Non-Conflicting Changes
	resultTyped, err := liveTyped.Merge(filteredDesired)
	if err != nil {
		return nil, err
	}

	// Convert back
	finalUnstructured, _ := converter.TypedToObject(resultTyped)
	return finalUnstructured.(*unstructured.Unstructured), nil
}
