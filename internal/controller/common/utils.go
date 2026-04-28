package common

import (
	"regexp"

	"github.com/miekg/dns"
)

func String(s *string, def string) string {
	if s == nil {
		return def
	}
	return *s
}

// fqdnRegex checks:
// 1. Total length is handled by the code logic.
// 2. Each label is 1-63 chars, starts/ends with alphanumeric, allows hyphens in middle.
// 3. TLD is at least 2 alpha characters (standard for most FQDNs).
var fqdnRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,63}\.?$`)

func IsValidFQDN(name string) bool {
	// 1. Basic length check (RFC 1035)
	if len(name) < 1 || len(name) > 253 {
		return false
	}

	// 2. Regex for character and label structure validation
	if !fqdnRegex.MatchString(name) {
		return false
	}

	// 3. Final safety check for the miekg/dns packer
	_, ok := dns.IsDomainName(name)
	return ok
}
