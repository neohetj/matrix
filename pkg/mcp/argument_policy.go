package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/neohetj/matrix/pkg/types"
)

// argumentPolicy 是 endpoint 独立的已编译规则，不持有可变配置切片。
type argumentPolicy struct {
	keys     map[string]struct{}
	prefixes []string
}

// compileArgumentPolicy 合并通用、模块及可信 Header 规则；模块不能关闭通用保护。
func compileArgumentPolicy(config *types.McpArgumentPolicy, contexts map[string]types.McpAuthContext) (*argumentPolicy, error) {
	if config == nil {
		return nil, errors.New("mcp endpoint argumentPolicy is required; declare {} for generic-only protection or configure module denyKeys/denyPrefixes")
	}
	policy := &argumentPolicy{keys: map[string]struct{}{}}
	for _, key := range []string{"authorization", "cookie", "company_id", "current_team_ids", "internal_token", "permissions", "roles", "session_id", "team_ids", "user_id"} {
		policy.keys[key] = struct{}{}
	}
	for i, key := range config.DenyKeys {
		normalized := normalizeSecurityKey(key)
		if normalized == "" {
			return nil, fmt.Errorf("mcp argumentPolicy.denyKeys[%d] must not be empty", i)
		}
		policy.keys[normalized] = struct{}{}
	}
	for i, prefix := range config.DenyPrefixes {
		normalized := normalizeSecurityKey(prefix)
		if normalized == "" {
			return nil, fmt.Errorf("mcp argumentPolicy.denyPrefixes[%d] must not be empty", i)
		}
		policy.prefixes = append(policy.prefixes, normalized)
	}
	for _, auth := range contexts {
		for header := range auth.Headers {
			if normalized := normalizeSecurityKey(header); normalized != "" {
				policy.keys[normalized] = struct{}{}
			}
		}
	}
	return policy, nil
}

// forbids 在 schema 和调用参数中使用相同的名称归一化与匹配规则。
func (p *argumentPolicy) forbids(key string) bool {
	normalized := normalizeSecurityKey(key)
	if _, ok := p.keys[normalized]; ok {
		return true
	}
	for _, prefix := range p.prefixes {
		if strings.HasPrefix(normalized, prefix) {
			return true
		}
	}
	return false
}

// validateInputSchema 编译 schema 并检查属性名，不把 schema 关键字当作调用参数。
func (p *argumentPolicy) validateInputSchema(toolName string, schema map[string]any) (*openapi3.Schema, error) {
	if len(schema) == 0 {
		return nil, nil
	}

	data, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("mcp tool %q inputSchema cannot be encoded: %w", toolName, err)
	}
	compiled := &openapi3.Schema{}
	if err := json.Unmarshal(data, compiled); err != nil {
		return nil, fmt.Errorf("mcp tool %q inputSchema is invalid: %w", toolName, err)
	}
	if err := compiled.Validate(context.Background()); err != nil {
		return nil, fmt.Errorf("mcp tool %q inputSchema is invalid: %w", toolName, err)
	}
	if forbidden := p.collectForbiddenSchemaKeys(compiled); len(forbidden) > 0 {
		return nil, fmt.Errorf("mcp tool %q inputSchema contains forbidden security context fields: %s", toolName, strings.Join(forbidden, ", "))
	}
	return compiled, nil
}

// collectForbiddenSchemaKeys 遍历已编译的属性、必填字段及嵌套组合 schema。
func (p *argumentPolicy) collectForbiddenSchemaKeys(schema *openapi3.Schema) []string {
	seen := map[string]struct{}{}
	visited := map[*openapi3.Schema]bool{}
	var walk func(*openapi3.Schema)
	var walkRef func(*openapi3.SchemaRef)
	walkRef = func(ref *openapi3.SchemaRef) {
		if ref != nil {
			walk(ref.Value)
		}
	}
	walk = func(s *openapi3.Schema) {
		if s == nil || visited[s] {
			return
		}
		visited[s] = true
		for key, child := range s.Properties {
			if p.forbids(key) {
				seen[key] = struct{}{}
			}
			walkRef(child)
		}
		for _, key := range s.Required {
			if p.forbids(key) {
				seen[key] = struct{}{}
			}
		}
		for _, group := range []openapi3.SchemaRefs{s.AllOf, s.AnyOf, s.OneOf} {
			for _, child := range group {
				walkRef(child)
			}
		}
		walkRef(s.Items)
		walkRef(s.Not)
		walkRef(s.AdditionalProperties.Schema)
	}
	walk(schema)
	out := make([]string, 0, len(seen))
	for key := range seen {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

// collectForbiddenKeys 递归检查 JSON 对象和数组，错误只包含字段名。
func (p *argumentPolicy) collectForbiddenKeys(value any) []string {
	seen := map[string]struct{}{}
	var walk func(any)
	walk = func(v any) {
		switch typed := v.(type) {
		case map[string]any:
			for key, child := range typed {
				if p.forbids(key) {
					seen[key] = struct{}{}
				}
				walk(child)
			}
		case []any:
			for _, child := range typed {
				walk(child)
			}
		}
	}
	walk(value)
	out := make([]string, 0, len(seen))
	for key := range seen {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

// rejectArguments 在解析身份与派发目标之前拒绝不可信身份字段。
func (p *argumentPolicy) rejectArguments(arguments map[string]any) error {
	forbidden := p.collectForbiddenKeys(arguments)
	if len(forbidden) > 0 {
		return fmt.Errorf("mcp tool arguments must not provide security context field %q", forbidden[0])
	}
	return nil
}

// normalizeSecurityKey 保持已有的大小写、首尾空白和分隔符归一化规则。
func normalizeSecurityKey(key string) string {
	key = strings.TrimSpace(strings.ToLower(key))
	key = strings.ReplaceAll(key, "-", "_")
	key = strings.ReplaceAll(key, ".", "_")
	key = strings.ReplaceAll(key, " ", "_")
	return key
}
