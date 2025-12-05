package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

var placeholderRegex = regexp.MustCompile(`\$\{([^}]+)\}`)

// ReplacePlaceholders replaces placeholders in a template string with values from a nested map.
// The placeholder format is ${path.to.value}.
func ReplacePlaceholders(template string, data map[string]any) string {
	if !strings.Contains(template, "${") || data == nil {
		return template
	}

	return placeholderRegex.ReplaceAllStringFunc(template, func(match string) string {
		// Extract the path from the placeholder, e.g., "dataT.objId.field" from "${dataT.objId.field}".
		path := strings.TrimSuffix(strings.TrimPrefix(match, "${"), "}")

		// Use the more robust ExtractByPath helper which can handle nested structs and maps.
		value, found, err := ExtractByPath(data, path)
		if err != nil || !found {
			// If an error occurs or the key is not found, return the original placeholder.
			// In a real application, you might want to log the error.
			return match
		}

		// Convert the found value to a string for replacement.
		return fmt.Sprintf("%v", value)
	})
}

// Ptr returns a pointer to the given value.
func Ptr[T any](v T) *T {
	return &v
}

// GenerateObjID generates a random object ID consisting of lowercase letters.
// The default length is 8 characters, which provides sufficient entropy for typical object IDs
// while remaining readable and url-safe.
func GenerateObjID(length int) string {
	if length <= 0 {
		length = 16
	}
	const letters = "abcdefghijklmnopqrstuvwxyz"
	ret := make([]byte, length)
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			// In the rare case of an error, fallback to a pseudo-random character or just 'a'
			// Ideally, handle error appropriately, but for simple ID generation, this might suffice
			// to avoid panics in utility functions.
			ret[i] = letters[0]
			continue
		}
		ret[i] = letters[num.Int64()]
	}
	return string(ret)
}
