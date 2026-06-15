package validation

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/neohetj/matrix/pkg/types"
)

func stringConfigValue(config types.ConfigMap, key string) string {
	if config == nil {
		return ""
	}
	value, ok := config[key]
	if !ok {
		return ""
	}
	if str, ok := value.(string); ok {
		return strings.TrimSpace(str)
	}
	return ""
}

func stringSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out[value] = struct{}{}
		}
	}
	return out
}

func functionDescriptorMap(functions []FunctionDescriptor) map[string]FunctionDescriptor {
	out := make(map[string]FunctionDescriptor, len(functions))
	for _, function := range functions {
		id := strings.TrimSpace(function.ID)
		if id == "" {
			continue
		}
		out[id] = function
	}
	return out
}

func mapValue(parent any, key string) (map[string]any, bool) {
	if parent == nil {
		return nil, false
	}
	var parentMap map[string]any
	switch typedParent := parent.(type) {
	case map[string]any:
		parentMap = typedParent
	case types.ConfigMap:
		parentMap = map[string]any(typedParent)
	default:
		return nil, false
	}
	value, ok := parentMap[key]
	if !ok {
		return nil, false
	}
	switch typed := value.(type) {
	case map[string]any:
		return typed, true
	case types.ConfigMap:
		return map[string]any(typed), true
	default:
		return nil, false
	}
}

func mapValueOrNil(parent any, key string) map[string]any {
	value, _ := mapValue(parent, key)
	return value
}

func sliceValue(parent any, key string) []any {
	if parent == nil {
		return nil
	}
	var parentMap map[string]any
	switch typedParent := parent.(type) {
	case map[string]any:
		parentMap = typedParent
	case types.ConfigMap:
		parentMap = map[string]any(typedParent)
	default:
		return nil
	}
	value, ok := parentMap[key]
	if !ok {
		return nil
	}
	if typed, ok := value.([]any); ok {
		return typed
	}
	return nil
}

func decodeConfig[T any](config types.ConfigMap) (T, error) {
	var out T
	if config == nil {
		return out, nil
	}
	data, err := json.Marshal(config)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return out, err
	}
	return out, nil
}

func collectConfigRefs(config types.ConfigMap) []string {
	refs := map[string]struct{}{}
	collectRefs(config, refs)
	return sortedStringKeys(refs)
}

func collectRefs(value any, refs map[string]struct{}) {
	switch typed := value.(type) {
	case types.ConfigMap:
		collectRefs(map[string]any(typed), refs)
	case map[string]any:
		for _, child := range typed {
			if refID, ok := refIDFromValue(child); ok && refID != "" {
				refs["ref://"+refID] = struct{}{}
			}
			collectRefs(child, refs)
		}
	case []any:
		for _, child := range typed {
			if refID, ok := refIDFromValue(child); ok && refID != "" {
				refs["ref://"+refID] = struct{}{}
			}
			collectRefs(child, refs)
		}
	}
}

func sortedStringKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		if strings.TrimSpace(key) != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func copyStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func providerName(provider types.ResourceProvider) string {
	if provider == nil {
		return ""
	}
	return provider.Name()
}

func appendPath(base string, key string) string {
	if base == "" {
		return key
	}
	return base + "." + key
}
