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

// PublicErrorDetails 返回可公开的结构化问题列表，不携带配置原值。
// PublicErrorDetails is safe for transports that opt in to structured errors.
func (p Issues) PublicErrorDetails() any {
	return struct {
		Issues Issues `json:"issues"`
	}{p}
}

// Error 拼接问题代码与配置键，供错误链记录安全摘要。
func (p Issues) Error() string {
	parts := make([]string, len(p))
	for i, v := range p {
		parts[i] = v.Code + ": " + v.Key
	}
	return strings.Join(parts, "; ")
}

// problem 构造带实例路径和 Schema 路径的标准问题。
func problem(key, code, schemaPath string) Issues {
	return Issues{{Key: key, Code: code, InstancePath: pointer(key), SchemaPath: schemaPath, Message: "configuration rejected: " + code}}
}

// pointer 将配置键转义为 JSON Pointer 路径片段。
func pointer(key string) string {
	if key == "" {
		return ""
	}
	return "/" + strings.ReplaceAll(strings.ReplaceAll(key, "~", "~0"), "/", "~1")
}

type ProfileBindingPolicy struct {
	Supported     bool                    `yaml:"supported" json:"supported"`
	ResourceKinds []string                `yaml:"resource_kinds,omitempty" json:"resource_kinds,omitempty"`
	Schemes       []string                `yaml:"schemes,omitempty" json:"schemes,omitempty"`
	ValueField    string                  `yaml:"value_field,omitempty" json:"value_field,omitempty"`
	BindingGroup  string                  `yaml:"binding_group,omitempty" json:"binding_group,omitempty"`
	Generation    *SecretGenerationPolicy `yaml:"generation,omitempty" json:"generation,omitempty"`
}

type SecretGenerationPolicy struct {
	Allowed  bool   `yaml:"allowed" json:"allowed"`
	Encoding string `yaml:"encoding" json:"encoding"`
	Bytes    int    `yaml:"bytes" json:"bytes"`
}

var schemePattern = regexp.MustCompile(`^[a-z][a-z0-9+.-]*$`)

// IsSupported 判断绑定策略是否显式启用，空策略视为关闭。
func (p *ProfileBindingPolicy) IsSupported() bool { return p != nil && p.Supported }

func (p *ProfileBindingPolicy) valueField() string {
	if p == nil || p.ValueField == "" {
		return "endpoint"
	}
	return p.ValueField
}

// ResolvedValueField returns the runtime resource field copied into the
// effective configuration. Older endpoint policies omit the field.
func (p *ProfileBindingPolicy) ResolvedValueField() string { return p.valueField() }

// Validate 校验绑定类型、资源种类和凭据策略之间的约束。
func (p *ProfileBindingPolicy) Validate(valueType string) error {
	if !p.IsSupported() {
		return nil
	}
	if valueType != "string" && valueType != "url" && valueType != "secret" {
		return fmt.Errorf("profile_binding requires a string, url, or secret configuration type")
	}
	if len(p.ResourceKinds) == 0 {
		return fmt.Errorf("profile_binding requires non-empty resource_kinds")
	}
	valueField := p.valueField()
	if valueField != "endpoint" && valueField != "secret" {
		return fmt.Errorf("profile_binding value_field must be endpoint or secret")
	}
	for _, v := range p.ResourceKinds {
		if v != "database" && v != "service" && v != "secret" {
			return fmt.Errorf("profile_binding resource_kinds must contain only database, service, or secret")
		}
	}
	if valueField == "secret" {
		if valueType != "secret" || len(p.ResourceKinds) != 1 || p.ResourceKinds[0] != "secret" || len(p.Schemes) != 0 {
			return fmt.Errorf("secret profile_binding requires secret type, secret resource kind, and no schemes")
		}
		if p.Generation != nil && (p.Generation.Encoding != "hex" || p.Generation.Bytes < 16 || p.Generation.Bytes > 128) {
			return fmt.Errorf("secret profile_binding generation requires hex encoding and 16-128 bytes")
		}
		return nil
	}
	if len(p.Schemes) == 0 || slices.Contains(p.ResourceKinds, "secret") || p.Generation != nil || p.BindingGroup != "" {
		return fmt.Errorf("endpoint profile_binding requires schemes and cannot use secret options")
	}
	for _, v := range p.Schemes {
		if !schemePattern.MatchString(v) {
			return fmt.Errorf("profile_binding schemes must be lowercase URI schemes")
		}
	}
	return nil
}

// Matches 判断资源种类与协议是否同时满足绑定策略。
func (p *ProfileBindingPolicy) Matches(kind, scheme string) bool {
	return p.IsSupported() && p.valueField() == "endpoint" && slices.Contains(p.ResourceKinds, kind) && slices.Contains(p.Schemes, scheme)
}

func (p *ProfileBindingPolicy) MatchesSecret(kind string) bool {
	return p.IsSupported() && p.valueField() == "secret" && kind == "secret" && slices.Contains(p.ResourceKinds, kind)
}

// Clone 复制定义及其可变集合，隔离调用方的修改。
func (p *ProfileBindingPolicy) Clone() *ProfileBindingPolicy {
	if p == nil {
		return nil
	}
	result := *p
	result.ResourceKinds = slices.Clone(p.ResourceKinds)
	result.Schemes = slices.Clone(p.Schemes)
	if p.Generation != nil {
		generation := *p.Generation
		result.Generation = &generation
	}
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

// Clone 复制定义及其可变集合，隔离调用方的修改。
func (f *Frozen) Clone() *Frozen {
	if f == nil {
		return nil
	}
	v := clone(*f)
	return &v
}

// clone 通过保留数值精度的 JSON 往返深拷贝内部定义。
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
