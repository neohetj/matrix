package utils

import (
	"fmt"
	"regexp"
	"strings"
)

var placeholderRegex = regexp.MustCompile(`\$\{([^}]+)\}`)

// ReplacePlaceholders replaces placeholders in a template string with values from a nested map.
// The placeholder format is ${path.to.value}.
func ReplacePlaceholders(template string, data map[string]interface{}) string {
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
