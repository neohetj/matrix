package mcp

import (
	"encoding/json"
	"regexp"
	"strings"
)

var secretJSONKeyPattern = regexp.MustCompile(`(?i)(authorization|cookie|access_token|refresh_token|id_token|client_secret|internal_token|api_key)$`)

// Sanitize decoded string leaves so escaped log newlines cannot be consumed
// as part of a credential. JSON structure and numeric metadata stay intact.
func sanitizeJSONOrText(text string) string {
	var value any
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	if json.Valid([]byte(text)) && decoder.Decode(&value) == nil {
		encoded, err := json.Marshal(redactJSON(value))
		if err == nil {
			return string(encoded)
		}
	}
	return redactPlainText(text)
}

func redactJSON(value any) any {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			if secretJSONKeyPattern.MatchString(strings.TrimSpace(key)) {
				v[key] = "[redacted]"
			} else {
				v[key] = redactJSON(child)
			}
		}
	case []any:
		for i := range v {
			v[i] = redactJSON(v[i])
		}
	case string:
		return redactPlainText(v)
	}
	return value
}

func redactPlainText(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		line = bearerTokenPattern.ReplaceAllString(line, "Bearer [redacted]")
		lines[i] = secretPairPattern.ReplaceAllString(line, "${1}${2}[redacted]")
	}
	return strings.Join(lines, "\n")
}
