// Package catalog loads immutable module configuration definitions and validates
// effective values. It never discovers workspaces or resolves deployment secrets.
package catalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

const Dialect = "https://json-schema.org/draft/2020-12/schema"
const Format = "matrix-catalog/2"

type Issue struct {
	Key          string `json:"key,omitempty"`
	InstancePath string `json:"instance_path"`
	SchemaPath   string `json:"schema_path"`
	Code         string `json:"code"`
	Message      string `json:"message"`
}
type Issues []Issue

// PublicErrorDetails is safe for transports that opt in to structured errors.
func (p Issues) PublicErrorDetails() any {
	return struct {
		Issues Issues `json:"issues"`
	}{p}
}

func (p Issues) Error() string {
	parts := make([]string, len(p))
	for i, v := range p {
		parts[i] = v.Code + ": " + v.Key
	}
	return strings.Join(parts, "; ")
}
func problem(key, code, schemaPath string) Issues {
	return Issues{{Key: key, Code: code, InstancePath: pointer(key), SchemaPath: schemaPath, Message: "configuration rejected: " + code}}
}
func pointer(key string) string {
	if key == "" {
		return ""
	}
	return "/" + strings.ReplaceAll(strings.ReplaceAll(key, "~", "~0"), "/", "~1")
}

type ProfileBindingPolicy struct {
	Supported     bool     `yaml:"supported" json:"supported"`
	ResourceKinds []string `yaml:"resource_kinds,omitempty" json:"resource_kinds,omitempty"`
	Schemes       []string `yaml:"schemes,omitempty" json:"schemes,omitempty"`
}

var schemePattern = regexp.MustCompile(`^[a-z][a-z0-9+.-]*$`)

func (p *ProfileBindingPolicy) IsSupported() bool { return p != nil && p.Supported }
func (p *ProfileBindingPolicy) Validate(valueType string) error {
	if !p.IsSupported() {
		return nil
	}
	if valueType != "string" && valueType != "url" && valueType != "secret" {
		return fmt.Errorf("profile_binding requires a string value")
	}
	if len(p.ResourceKinds) == 0 || len(p.Schemes) == 0 {
		return fmt.Errorf("profile_binding requires resource_kinds and schemes")
	}
	for _, v := range p.ResourceKinds {
		if v != "database" && v != "service" {
			return fmt.Errorf("profile_binding resource kind unsupported")
		}
	}
	for _, v := range p.Schemes {
		if !schemePattern.MatchString(v) {
			return fmt.Errorf("profile_binding scheme unsupported")
		}
	}
	return nil
}
func (p *ProfileBindingPolicy) Matches(kind, scheme string) bool {
	return p.IsSupported() && p.Validate("string") == nil && slices.Contains(p.ResourceKinds, kind) && slices.Contains(p.Schemes, scheme)
}
func (p *ProfileBindingPolicy) Clone() *ProfileBindingPolicy {
	if p == nil {
		return nil
	}
	result := *p
	result.ResourceKinds = slices.Clone(p.ResourceKinds)
	result.Schemes = slices.Clone(p.Schemes)
	return &result
}

type Item struct {
	Key            string                `yaml:"key" json:"key"`
	Owner          string                `yaml:"owner" json:"owner"`
	Type           string                `yaml:"type" json:"type"`
	Description    string                `yaml:"description" json:"description"`
	Resolution     string                `yaml:"resolution" json:"resolution"`
	Required       bool                  `yaml:"required,omitempty" json:"required,omitempty"`
	Default        any                   `yaml:"default,omitempty" json:"default,omitempty"`
	Secret         bool                  `yaml:"secret" json:"secret"`
	Aliases        []string              `yaml:"aliases,omitempty" json:"aliases,omitempty"`
	Consumers      []string              `yaml:"consumers,omitempty" json:"consumers,omitempty"`
	Deprecated     bool                  `yaml:"deprecated,omitempty" json:"deprecated,omitempty"`
	ProfileBinding *ProfileBindingPolicy `yaml:"profile_binding,omitempty" json:"profile_binding,omitempty"`
	Schema         map[string]any        `yaml:"schema,omitempty" json:"schema,omitempty"`
}
type Definition struct {
	Name     string         `yaml:"-" json:"name"`
	Version  string         `yaml:"version" json:"version"`
	Module   string         `yaml:"module" json:"module"`
	Domain   string         `yaml:"domain" json:"domain"`
	Items    []Item         `yaml:"items" json:"items"`
	Schema   map[string]any `yaml:"schema,omitempty" json:"schema,omitempty"`
	UISchema map[string]any `yaml:"ui_schema,omitempty" json:"ui_schema,omitempty"`
}

// Frozen contains definitions only, never resolved values. Consumers must check
// its format and digest, not replace it with definitions from a newer commit.
type Frozen struct {
	Format    string       `json:"format"`
	Digest    string       `json:"digest"`
	Documents []Definition `json:"documents"`
}

// UnmarshalJSON 保留冻结定义中 default、enum 和数值边界的十进制精度。
func (f *Frozen) UnmarshalJSON(data []byte) error {
	type frozenJSON Frozen
	var value frozenJSON
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	*f = Frozen(value)
	return nil
}

func (f *Frozen) Clone() *Frozen {
	if f == nil {
		return nil
	}
	v := clone(*f)
	return &v
}
func clone[T any](v T) T {
	encoded, err := json.Marshal(v)
	if err != nil {
		panic("catalog: non-JSON internal definition")
	}
	var copy T
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	if decoder.Decode(&copy) != nil {
		panic("catalog: invalid internal definition")
	}
	return copy
}
