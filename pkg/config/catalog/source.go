package catalog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Document struct {
	Name    string
	Content []byte
}

// CatalogSource must return one consistent version of the complete document set.
type CatalogSource interface {
	Documents(context.Context) ([]Document, error)
}
type Documents []Document

func (d Documents) Documents(ctx context.Context) ([]Document, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return d, nil
}

// FSSource accepts embed.FS, os.DirFS(explicitDirectory), or an in-memory FS.
// It reads only *_catalog.yaml at the supplied FS root; no directory probing.
type FSSource struct{ FS fs.FS }

func (s FSSource) Documents(ctx context.Context) ([]Document, error) {
	if s.FS == nil {
		return nil, problem("", "catalog_source", "")
	}
	names, err := fs.Glob(s.FS, "*_catalog.yaml")
	if err != nil {
		return nil, problem("", "catalog_source", "")
	}
	var result []Document
	for _, name := range names {
		if ctx.Err() != nil {
			return nil, problem("", "catalog_source", "")
		}
		f, err := s.FS.Open(name)
		if err != nil {
			return nil, problem("", "catalog_source", "")
		}
		raw, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
		f.Close()
		if err != nil || len(raw) > 1<<20 {
			return nil, problem("", "catalog_source", "")
		}
		result = append(result, Document{name, raw})
	}
	return result, nil
}

// Decode performs strict document checks; cross-document rules compile in Load.
func Decode(d Document) (Definition, error) {
	var v Definition
	if len(d.Content) > 1<<20 {
		return v, problem("", "catalog_size", "")
	}
	decoder := yaml.NewDecoder(bytes.NewReader(d.Content))
	decoder.KnownFields(true)
	if err := decoder.Decode(&v); err != nil {
		return v, problem("", "catalog_decode", "")
	}
	var trailing any
	if decoder.Decode(&trailing) != io.EOF {
		return v, problem("", "catalog_decode", "")
	}
	v.Name = d.Name
	for i := range v.Items {
		v.Items[i].Key = strings.TrimSpace(v.Items[i].Key)
		// v1 modules transported list-like settings as strings and split them in
		// module code. Preserve that wire format when the declared list default is
		// textual; v2 requires an actual array.
		if v.Version == "1" && v.Items[i].Type == "string_list" {
			if _, ok := v.Items[i].Default.(string); ok {
				v.Items[i].Type = "string"
			}
		}
		// v1 catalogs used empty strings/JSON collections as absence markers. Drop
		// these before validation/freezing; they never become secret values.
		if v.Version == "1" && v.Items[i].Secret {
			if value, ok := v.Items[i].Default.(string); ok && (value == "" || value == "{}" || value == "[]") {
				v.Items[i].Default = nil
			}
		}
	}
	if err := checkDefinition(v); err != nil {
		return Definition{}, err
	}
	return v, nil
}
func Load(ctx context.Context, source CatalogSource) (*Catalog, error) {
	if source == nil {
		return nil, problem("", "catalog_source", "")
	}
	docs, err := source.Documents(ctx)
	if err != nil {
		return nil, problem("", "catalog_source", "")
	}
	definitions := make([]Definition, 0, len(docs))
	for _, doc := range docs {
		v, err := Decode(doc)
		if err != nil {
			return nil, err
		}
		definitions = append(definitions, v)
	}
	return compileDefinitions(definitions)
}
func checkDefinition(d Definition) error {
	if (d.Version != "1" && d.Version != "2") || strings.TrimSpace(d.Module) == "" || strings.TrimSpace(d.Domain) == "" || d.Items == nil {
		return problem("", "catalog_identity", "")
	}
	if d.Version == "1" && (d.Schema != nil || d.UISchema != nil) {
		return problem("", "catalog_version", "")
	}
	if d.UISchema != nil {
		if err := checkUISchema(d.UISchema, "/ui_schema", 0); err != nil {
			return err
		}
	}
	for _, item := range d.Items {
		if strings.TrimSpace(item.Key) == "" || strings.TrimSpace(item.Owner) == "" || strings.TrimSpace(item.Description) == "" {
			return problem(item.Key, "catalog_item", "")
		}
		if item.Resolution != "placeholder" && item.Resolution != "node_explicit" {
			return problem(item.Key, "catalog_resolution", "")
		}
		if _, ok := schemaTypes[item.Type]; !ok {
			return problem(item.Key, "catalog_type", "")
		}
		if item.Secret && item.Default != nil {
			return problem(item.Key, "secret_default", "")
		}
		if item.Type == "secret" && !item.Secret {
			return problem(item.Key, "secret_policy", "")
		}
		if err := item.ProfileBinding.Validate(item.Type); err != nil {
			return problem(item.Key, "profile_binding", "")
		}
		if d.Version == "1" && item.Schema != nil {
			return problem(item.Key, "catalog_version", "")
		}
	}
	return nil
}

// checkUISchema accepts JSON Forms presentation metadata, but deliberately
// constrains behavioral rules to SHOW/HIDE. UI metadata never changes the
// resolved, validated, or injected configuration value.
func checkUISchema(node map[string]any, path string, depth int) error {
	if depth > 64 {
		return problem("", "ui_schema_depth", path)
	}
	for _, forbidden := range []string{"runtime", "runtime_when", "active_when", "required_when"} {
		if _, ok := node[forbidden]; ok {
			return problem("", "ui_schema_semantics", path+pointer(forbidden))
		}
	}
	if raw, ok := node["scope"]; ok {
		if !validUIScope(raw) {
			return problem("", "ui_schema_scope", path+"/scope")
		}
	}
	if raw, ok := node["elements"]; ok {
		elements, ok := raw.([]any)
		if !ok {
			return problem("", "ui_schema_shape", path+"/elements")
		}
		for i, rawElement := range elements {
			element, ok := rawElement.(map[string]any)
			if !ok {
				return problem("", "ui_schema_shape", fmt.Sprintf("%s/elements/%d", path, i))
			}
			if err := checkUISchema(element, fmt.Sprintf("%s/elements/%d", path, i), depth+1); err != nil {
				return err
			}
		}
	}
	rawRule, ok := node["rule"]
	if !ok {
		return nil
	}
	rule, ok := rawRule.(map[string]any)
	if !ok {
		return problem("", "ui_schema_rule", path+"/rule")
	}
	for key := range rule {
		if key != "effect" && key != "condition" {
			return problem("", "ui_schema_rule", path+"/rule"+pointer(key))
		}
	}
	effect, ok := rule["effect"].(string)
	if !ok || (effect != "SHOW" && effect != "HIDE") {
		return problem("", "ui_schema_effect", path+"/rule/effect")
	}
	condition, ok := rule["condition"].(map[string]any)
	if !ok {
		return problem("", "ui_schema_condition", path+"/rule/condition")
	}
	for key := range condition {
		if key != "scope" && key != "schema" && key != "failWhenUndefined" {
			return problem("", "ui_schema_condition", path+"/rule/condition"+pointer(key))
		}
	}
	if !validUIScope(condition["scope"]) {
		return problem("", "ui_schema_scope", path+"/rule/condition/scope")
	}
	conditionSchema, ok := condition["schema"]
	if !ok {
		return problem("", "ui_schema_condition", path+"/rule/condition/schema")
	}
	if err := checkSchema(conditionSchema, "", path+"/rule/condition/schema", depth+1); err != nil {
		return err
	}
	if fail, ok := condition["failWhenUndefined"]; ok {
		if _, valid := fail.(bool); !valid {
			return problem("", "ui_schema_condition", path+"/rule/condition/failWhenUndefined")
		}
	}
	return nil
}

func validUIScope(raw any) bool {
	scope, ok := raw.(string)
	return ok && strings.HasPrefix(scope, "#/properties/") && len(scope) > len("#/properties/")
}

func orderedDefinitions(defs []Definition) []Definition {
	result := clone(defs)
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}
