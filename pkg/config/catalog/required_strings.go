package catalog

import "sort"

// requireNonEmptyStrings 在已复制的 Schema 断言分支中补齐必填字符串约束，不重写 if/not 谓词。
func requireNonEmptyStrings(raw any, catalogProperties map[string]any) {
	rule, ok := raw.(map[string]any)
	if !ok {
		return
	}
	for _, keyword := range []string{"allOf", "anyOf", "oneOf"} {
		children, _ := rule[keyword].([]any)
		for _, child := range children {
			requireNonEmptyStrings(child, catalogProperties)
		}
	}
	for _, keyword := range []string{"then", "else"} {
		requireNonEmptyStrings(rule[keyword], catalogProperties)
	}
	if dependencies, ok := rule["dependentSchemas"].(map[string]any); ok {
		for _, child := range dependencies {
			requireNonEmptyStrings(child, catalogProperties)
		}
	}

	var extra []any
	if fields := requiredStringFields(rule["required"], catalogProperties); len(fields) > 0 {
		extra = append(extra, map[string]any{"properties": fields})
	}
	if dependencies, ok := rule["dependentRequired"].(map[string]any); ok {
		keys := make([]string, 0, len(dependencies))
		for key := range dependencies {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if fields := requiredStringFields(dependencies[key], catalogProperties); len(fields) > 0 {
				extra = append(extra, map[string]any{
					"if":   map[string]any{"required": []any{key}},
					"then": map[string]any{"properties": fields},
				})
			}
		}
	}
	if len(extra) > 0 {
		all, _ := rule["allOf"].([]any)
		rule["allOf"] = append(all, extra...)
	}
}

// requiredStringFields 仅为当前分支要求的 Catalog 字符串生成约束，可选字段、false 和零不受影响。
func requiredStringFields(raw any, catalogProperties map[string]any) map[string]any {
	fields := map[string]any{}
	keys, _ := raw.([]any)
	for _, rawKey := range keys {
		key, ok := rawKey.(string)
		if !ok {
			continue
		}
		property, _ := catalogProperties[key].(map[string]any)
		if property["type"] == "string" {
			fields[key] = map[string]any{"minLength": 1}
		}
	}
	return fields
}
