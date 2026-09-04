package catalog

// checkRuleOwnership keeps Catalog types as the single source of truth and
// rejects typos in references to configuration keys. Nested array elements are
// ordinary JSON Schema values; their types do not redefine a Catalog item.
func checkRuleOwnership(raw any, properties map[string]any, key string, root bool) error {
	m, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	if value, hasType := m["type"]; hasType {
		if !root && key != "" {
			return problem(key, "catalog_owned_keyword", "/type")
		}
		if root && value != "object" {
			return problem("", "catalog_owned_keyword", "/type")
		}
	}
	if root {
		if props, ok := m["properties"].(map[string]any); ok {
			for name, child := range props {
				if _, found := properties[name]; !found {
					return problem(name, "schema_unknown_key", "/properties"+pointer(name))
				}
				if err := checkRuleOwnership(child, properties, name, false); err != nil {
					return err
				}
			}
		}
		if required, ok := m["required"].([]any); ok {
			for _, rawName := range required {
				if name, ok := rawName.(string); ok {
					if _, found := properties[name]; !found {
						return problem(name, "schema_unknown_key", "/required")
					}
				}
			}
		}
		if deps, ok := m["dependentRequired"].(map[string]any); ok {
			for name, keys := range deps {
				if _, found := properties[name]; !found {
					return problem(name, "schema_unknown_key", "/dependentRequired")
				}
				if err := checkRuleOwnership(map[string]any{"required": keys}, properties, "", true); err != nil {
					return err
				}
			}
		}
		if deps, ok := m["dependentSchemas"].(map[string]any); ok {
			for name, child := range deps {
				if _, found := properties[name]; !found {
					return problem(name, "schema_unknown_key", "/dependentSchemas")
				}
				if err := checkRuleOwnership(child, properties, "", true); err != nil {
					return err
				}
			}
		}
	}
	for _, keyword := range []string{"if", "then", "else", "not"} {
		if err := checkRuleOwnership(m[keyword], properties, key, root); err != nil {
			return err
		}
	}
	for _, keyword := range []string{"allOf", "anyOf", "oneOf"} {
		if list, ok := m[keyword].([]any); ok {
			for _, child := range list {
				if err := checkRuleOwnership(child, properties, key, root); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
