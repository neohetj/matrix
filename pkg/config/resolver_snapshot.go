package config

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/neohetj/matrix/pkg/types"
)

type snapshotValues map[string]any

// Lookup 只读取已冻结来源，保留 false 和 0 的存在性。
func (v snapshotValues) Lookup(ctx context.Context, key string) (any, bool, error) {
	if ctx != nil && ctx.Err() != nil {
		return nil, false, fmt.Errorf("configuration context canceled")
	}
	value, found := v[key]
	return value, found, nil
}

// Snapshot 仅冻结声明中的 key/alias，Secret 不读取 business 来源。
func (r *ConfigResolver) Snapshot(specs []ConfigSpec) (*ConfigResolver, error) {
	if r == nil {
		return nil, fmt.Errorf("configuration resolver missing")
	}
	env := &snapshotCollector{resolver: r, scope: "env", seen: map[string]bool{}, values: snapshotValues{}}
	business := &snapshotCollector{resolver: r, scope: "engine", seen: map[string]bool{}, values: snapshotValues{}}
	collector := NewConfigResolver(WithValueSources(context.Background(), env, business))
	for _, spec := range specs {
		// 构造时仅冻结实际会访问的来源，不执行字段必填或读取已被 env 遮蔽的 business。
		spec.Required = false
		if spec.Secret {
			spec.Resolution = ResolutionPlaceholder
		}
		if _, _, err := Resolve[any](collector, spec); err != nil {
			return nil, fmt.Errorf("configuration source_read: %s", spec.Key)
		}
	}
	return NewConfigResolver(WithValueSources(context.Background(), env.values, business.values)), nil
}

type snapshotCollector struct {
	resolver *ConfigResolver
	scope    string
	seen     map[string]bool
	values   snapshotValues
}

// Lookup 为同一来源 key 冻结一次读取结果，包括不存在的结果。
func (s *snapshotCollector) Lookup(ctx context.Context, key string) (any, bool, error) {
	if s.seen[key] {
		return s.values.Lookup(ctx, key)
	}
	s.seen[key] = true
	raw, found, err := s.resolver.resolveAssetRaw(key, s.scope)
	if err != nil || !found {
		return nil, found, err
	}
	value, ok := copySnapshotValue(raw)
	if !ok {
		return nil, false, fmt.Errorf("configuration source type unsupported")
	}
	s.values[key] = value
	return value, true, nil
}

// copySnapshotValue 深拷贝支持的配置数据，不通过 JSON 损失整数精度。
func copySnapshotValue(raw any) (any, bool) {
	switch value := raw.(type) {
	case nil, string, json.Number, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return value, true
	case []string:
		return append([]string(nil), value...), true
	case []any:
		result := make([]any, len(value))
		for i, entry := range value {
			var ok bool
			result[i], ok = copySnapshotValue(entry)
			if !ok {
				return nil, false
			}
		}
		return result, true
	case types.ConfigMap:
		return copySnapshotValue(map[string]any(value))
	case map[string]any:
		result := make(map[string]any, len(value))
		for key, entry := range value {
			var ok bool
			result[key], ok = copySnapshotValue(entry)
			if !ok {
				return nil, false
			}
		}
		return result, true
	default:
		return nil, false
	}
}
