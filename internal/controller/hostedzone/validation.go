package hostedzone

import (
	"fmt"

	"github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
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
	if len(r.Object.Spec.DomainNames) == 0 {
		return v1alpha1.ReasonDomainNameMissing, fmt.Errorf("at one domain name required")
	}
	if r.Object.Spec.EMail == "" {
		return v1alpha1.ReasonEMailMissing, fmt.Errorf("email address required")
	}
	if r.Object.Spec.Expire == 0 {
		return v1alpha1.ReasonExpireMissing, fmt.Errorf("expire required")
	}
	if r.Object.Spec.Refresh == 0 || r.Object.Spec.Retry == 0 || r.Object.Spec.MinimumTTL == 0 {
		return v1alpha1.ReasonTTLMissing, fmt.Errorf("refresh, retry or minimumTTL required")
	}

	if r.Object.Spec.ParentRef != "" {
		if r.Object.Spec.Runtime != nil {
			return v1alpha1.ReasonInvalidNesting, fmt.Errorf("runtime set for nested zone")
		}
		if r.Object.Spec.Class != nil {
			return v1alpha1.ReasonInvalidNesting, fmt.Errorf("class set for nested zone")
		}
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
