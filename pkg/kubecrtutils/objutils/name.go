package objutils

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/mandelsoft/goutils/general"
)

const MAX_NAMELEN = 253
const MAX_NAMESPACELEN = 63

func GenerateUniqueName(prefix, namespace, name string, length ...int) string {
	// 1. Create a hash of the unique parts
	var fullInput string
	if namespace == "" {
		fullInput = name

	} else {
		fullInput = fmt.Sprintf("%s-%s", namespace, name)
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(fullInput)))[:8] // Take first 8 chars

	// 2. Sanitize and truncate the name
	// We want: prefix (variable) + name (truncated) + "-" + hash (8 chars)
	// Total must be <= 63 for safe usage across all resource types
	limit := general.OptionalDefaulted(MAX_NAMELEN, length...)
	maxNameLen := limit - len(prefix) - len(hash) - 2 // -2 for the separators

	shortName := fullInput
	if len(shortName) > maxNameLen {
		shortName = fullInput[:maxNameLen]
	}

	// Clean up trailing dashes if truncation cut in the middle
	shortName = strings.TrimSuffix(shortName, "-")

	return fmt.Sprintf("%s-%s-%s", prefix, shortName, hash)
}
