package hostedzone

import (
	"fmt"

	"github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/common"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func String(s *string, def string) string {
	if s == nil {
		return def
	}
	return *s
}

func IsASCIIAlnum(b byte) bool {
	return (b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9')
}

func SetField[T comparable](target *T, value T) bool {
	if *target == value {
		return false
	}
	*target = value
	return true
}

// Comparable must handle nil pointers.
type Comparable[T any] interface {
	*T
	Equals(other *T) bool
}

func SetPointerField[T any, P Comparable[T]](target *P, value P) bool {
	if *target == value {
		return false
	}
	if *target != nil {
		if !(*target).Equals(value) {
			return false
		}
	}
	*target = value
	return true
}

func IsASCIIAlnumString(s string) bool {
	if s == "" {
		return true
	}
	for i := 0; i < len(s); i++ {
		if !IsASCIIAlnum(s[i]) {
			return false
		}
	}
	return true
}

func (r *ReconcileRequest) Validate(root *Responsibility) (string, error) {
	reason, err := common.ValidateZone(r.Object)
	if err != nil {
		return reason, err
	}

	if root.Parent != nil {
		c := meta.FindStatusCondition(root.Parent.Status.Conditions, v1alpha1.ValidationConditionType)
		if c != nil {
			if c.Status != metav1.ConditionTrue {
				return v1alpha1.ReasonInvalidParent, fmt.Errorf("%s: %s", root.Parent.Name, c.Message)
			}
		}

	}
	return v1alpha1.ReasonConfigurarationValid, nil
}
