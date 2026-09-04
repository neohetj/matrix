package catalog

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
)

var schemaTypes = map[string]string{"string": "string", "bool": "boolean", "int": "integer", "int64": "integer", "float": "number", "duration": "string", "url": "string", "path": "string", "ref": "string", "string_list": "array", "secret": "string"}

type Catalog struct {
	documents []Definition
	items     []Item
	schema    *jsonschema.Schema
	fields    map[string]*jsonschema.Schema
	digest    string
}

func (c *Catalog) Items() []Item { return clone(c.items) }
func (c *Catalog) Item(key string) (Item, bool) {
	for _, item := range c.items {
		if item.Key == key {
			return clone(item), true
		}
	}
	return Item{}, false
}
func (c *Catalog) Freeze() *Frozen {
	return &Frozen{Format: Format, Digest: c.digest, Documents: clone(c.documents)}
}
func Restore(f *Frozen) (*Catalog, error) {
	if f == nil || f.Format != Format {
		return nil, problem("", "catalog_format", "")
	}
	c, err := compileDefinitions(f.Documents)
	if err != nil {
		return nil, err
	}
	if c.digest != f.Digest {
		return nil, problem("", "catalog_digest", "")
	}
	return c, nil
}

type noExternalLoader struct{}

func (noExternalLoader) Load(string) (any, error) {
	return nil, fmt.Errorf("external schema references disabled")
}

func compileDefinitions(defs []Definition) (*Catalog, error) {
	if len(defs) == 0 {
		return nil, problem("", "catalog_empty", "")
	}
	// Reject non-JSON metadata before cloning; errors must not echo input values.
	if _, err := json.Marshal(defs); err != nil {
		return nil, problem("", "catalog_json", "")
	}
	c := &Catalog{documents: orderedDefinitions(defs)}
	props := map[string]any{}
	required := []string{}
	all := []any{}
	names := map[string]bool{}
	module := c.documents[0].Module
	for _, d := range c.documents {
		if err := checkDefinition(d); err != nil {
			return nil, err
		}
		if d.Module != module || names[d.Name] {
			return nil, problem("", "catalog_identity", "")
		}
		names[d.Name] = true
		for _, item := range d.Items {
			if _, ok := props[item.Key]; ok {
				return nil, problem(item.Key, "duplicate_key", "")
			}
			base := map[string]any{"type": schemaTypes[item.Type]}
			if item.Type == "string_list" {
				base["items"] = map[string]any{"type": "string"}
			}
			if item.Schema != nil {
				if _, duplicate := item.Schema["type"]; duplicate {
					return nil, problem(item.Key, "catalog_owned_keyword", "/type")
				}
				if err := checkSchema(item.Schema, item.Key, "", 0); err != nil {
					return nil, err
				}
				base["allOf"] = []any{item.Schema}
			}
			props[item.Key] = base
			if item.Required {
				required = append(required, item.Key)
			}
			c.items = append(c.items, item)
		}
		if d.Schema != nil {
			if err := checkSchema(d.Schema, "", "", 0); err != nil {
				return nil, err
			}
			all = append(all, d.Schema)
		}
	}
	sort.Slice(c.items, func(i, j int) bool { return c.items[i].Key < c.items[j].Key })
	for _, d := range c.documents {
		if err := checkRuleOwnership(d.Schema, props, "", true); err != nil {
			return nil, err
		}
		for _, item := range d.Items {
			if err := checkRuleOwnership(item.Schema, props, item.Key, false); err != nil {
				return nil, err
			}
		}
	}
	base := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		base["required"] = required
	}
	all = append([]any{base}, all...)
	root := map[string]any{"$schema": Dialect, "allOf": all}
	// JSON normalization gives the schema library its documented JSON value types.
	root = clone(root)
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.UseLoader(noExternalLoader{})
	if compiler.AddResource("urn:matrix:catalog", root) != nil {
		return nil, problem("", "schema_compile", "")
	}
	var err error
	c.schema, err = compiler.Compile("urn:matrix:catalog")
	if err != nil {
		return nil, problem("", "schema_compile", "")
	}
	c.fields = map[string]*jsonschema.Schema{}
	for _, item := range c.items {
		field, err := compiler.Compile("urn:matrix:catalog#/allOf/0/properties" + pointer(item.Key))
		if err != nil {
			return nil, problem(item.Key, "schema_compile", "")
		}
		c.fields[item.Key] = field
		if item.Default != nil {
			value, err := Convert(item, item.Default)
			if err != nil {
				return nil, err
			}
			if field.Validate(clone(value)) != nil {
				return nil, problem(item.Key, "catalog_default", "")
			}
		}
	}
	encoded, _ := json.Marshal(c.documents)
	sum := sha256.Sum256(encoded)
	c.digest = "sha256:" + hex.EncodeToString(sum[:])
	return c, nil
}

// Supported dialect: 2020-12 validation subset below. References (including
// local $ref), custom vocabularies, format/content and default are intentionally
// rejected, not ignored. No filesystem/network reference resolution is allowed.
// ui_schema is checked separately as JSON Forms presentation metadata. It is
// not a configuration validation, resolution, injection, or security policy.
func checkSchema(raw any, key, path string, depth int) error {
	if depth > 64 {
		return problem(key, "schema_depth", path)
	}
	if _, ok := raw.(bool); ok {
		return nil
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return problem(key, "schema_shape", path)
	}
	for keyword, value := range m {
		p := path + pointer(keyword)
		switch keyword {
		case "$schema":
			if value != Dialect {
				return problem(key, "schema_dialect", p)
			}
		case "type", "enum", "const", "required", "minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum", "multipleOf", "minLength", "maxLength", "pattern", "minItems", "maxItems", "uniqueItems", "minContains", "maxContains", "minProperties", "maxProperties", "dependentRequired", "title", "description":
		case "allOf", "anyOf", "oneOf", "prefixItems":
			list, ok := value.([]any)
			if !ok {
				return problem(key, "schema_shape", p)
			}
			for i, child := range list {
				if err := checkSchema(child, key, fmt.Sprintf("%s/%d", p, i), depth+1); err != nil {
					return err
				}
			}
		case "if", "then", "else", "not", "items", "contains", "additionalProperties", "propertyNames":
			if err := checkSchema(value, key, p, depth+1); err != nil {
				return err
			}
		case "properties", "patternProperties", "dependentSchemas":
			children, ok := value.(map[string]any)
			if !ok {
				return problem(key, "schema_shape", p)
			}
			for name, child := range children {
				if err := checkSchema(child, key, p+pointer(name), depth+1); err != nil {
					return err
				}
			}
		default:
			return problem(key, "schema_keyword_unsupported", p)
		}
	}
	return nil
}

// Validate validates a complete, already resolved and typed configuration. It
// never applies defaults, mutates input, or interprets secret binding tokens.
func (c *Catalog) Validate(values map[string]any) Issues {
	encoded, err := json.Marshal(values)
	if err != nil {
		return problem("", "value_type", "")
	}
	var normalized any
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	if decoder.Decode(&normalized) != nil {
		return problem("", "value_type", "")
	}
	err = c.schema.Validate(normalized)
	return validationIssues(err)
}

// ValidateProvided checks supplied fields only. It is a draft gate, never an
// execution gate: root required/conditional rules remain intact in Validate.
func (c *Catalog) ValidateProvided(values map[string]any) Issues {
	var result Issues
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		item, ok := c.Item(key)
		if !ok {
			result = append(result, problem(key, "unknown_key", "")...)
			continue
		}
		if raw, ok := values[key].(string); ok && raw == "" {
			continue // 与 resolver 一致：空字符串表示未提供，执行门禁检查完整性。
		}
		value, err := Convert(item, values[key])
		if err != nil {
			result = append(result, problem(key, "value_type", "")...)
			continue
		}
		issues := validationIssues(c.fields[key].Validate(clone(value)))
		for i := range issues {
			issues[i].Key = key
			issues[i].InstancePath = pointer(key) + issues[i].InstancePath
		}
		result = append(result, issues...)
	}
	return result
}

func validationIssues(err error) Issues {
	if err == nil {
		return nil
	}
	v, ok := err.(*jsonschema.ValidationError)
	if !ok {
		return problem("", "schema_validation", "")
	}
	var result Issues
	var walk func(*jsonschema.ValidationError)
	walk = func(e *jsonschema.ValidationError) {
		if len(e.Causes) > 0 {
			for _, child := range e.Causes {
				walk(child)
			}
			return
		}
		code := "schema_validation"
		kp := e.ErrorKind.KeywordPath()
		if len(kp) > 0 {
			code = kp[0]
		}
		instance := ""
		key := ""
		for _, token := range e.InstanceLocation {
			instance += pointer(token)
		}
		if len(e.InstanceLocation) > 0 {
			key = e.InstanceLocation[0]
		}
		schemaPath := strings.TrimPrefix(e.SchemaURL, "urn:matrix:catalog")
		for _, token := range kp {
			schemaPath += pointer(token)
		}
		if required, ok := e.ErrorKind.(*kind.Required); ok {
			for _, missing := range required.Missing {
				result = append(result, Issue{Key: missing, InstancePath: instance + pointer(missing), SchemaPath: schemaPath, Code: "required", Message: "required configuration is missing"})
			}
			return
		}
		result = append(result, Issue{Key: key, InstancePath: instance, SchemaPath: schemaPath, Code: code, Message: "configuration does not satisfy schema"})
	}
	walk(v)
	sort.Slice(result, func(i, j int) bool {
		a, b := result[i], result[j]
		return a.InstancePath+a.SchemaPath < b.InstancePath+b.SchemaPath
	})
	return result
}
