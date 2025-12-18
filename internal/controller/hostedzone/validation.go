package hostedzone

import (
	"fmt"
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

func (r *ReconcileRequest) Validate() (string, error) {
	if len(r.instance.Spec.DomainNames) == 0 {
		return ReasonConfigurarationValid, fmt.Errorf("at one domain name required")
	}
	if r.instance.Spec.EMail == "" {
		return ReasonEMailMissing, fmt.Errorf("email address required")
	}
	if r.instance.Spec.Expire == 0 {
		return ReasonExpireMissing, fmt.Errorf("expire required")
	}
	if r.instance.Spec.Refresh == 0 || r.instance.Spec.Retry == 0 || r.instance.Spec.MinimumTTL == 0 {
		return ReasonTTLMissing, fmt.Errorf("refresh, retry or minimumTTL required")
	}

	if r.instance.Spec.ParentRef != "" {
		if r.instance.Spec.Runtime != nil {
			return ReasonInvalidNesting, fmt.Errorf("runtime set for nested zone")
		}
		if r.instance.Spec.Class != nil {
			return ReasonInvalidNesting, fmt.Errorf("class set for nested zone")
		}
	}
	return ReasonConfigurarationValid, nil
}
