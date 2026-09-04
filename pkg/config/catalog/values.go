package catalog

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/neohetj/matrix/pkg/cnst"
	matrixconfig "github.com/neohetj/matrix/pkg/config"
	"github.com/neohetj/matrix/pkg/utils"
)

type ValueSource = matrixconfig.ConfigValueSource
type Values map[string]any

func (v Values) Lookup(ctx context.Context, key string) (any, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	value, found := v[key]
	return value, found, nil
}

// Explicit is used only for node_explicit definitions in whole-config resolution.
// A node-local override belongs to ResolveString, never to this global view.
type Sources struct{ Env, Business, Explicit ValueSource }
type View interface{ Lookup(string) (any, bool) }
type Check func(View) Issues
type Resolved struct{ values map[string]any }

func (v Resolved) Values() map[string]any {
	result := make(map[string]any, len(v.values))
	for k, value := range v.values {
		result[k] = copyValue(value)
	}
	return result
}
func (v Resolved) Lookup(key string) (any, bool) {
	value, found := v.values[key]
	return copyValue(value), found
}
func copyValue(v any) any {
	switch value := v.(type) {
	case []string:
		return append([]string(nil), value...)
	case []any:
		result := make([]any, len(value))
		for i, x := range value {
			result[i] = copyValue(x)
		}
		return result
	case map[string]any:
		result := map[string]any{}
		for k, x := range value {
			result[k] = copyValue(x)
		}
		return result
	default:
		return v
	}
}
func spec(item Item) matrixconfig.ConfigSpec {
	return matrixconfig.ConfigSpec{Key: item.Key, Type: cnst.STRING, Resolution: matrixconfig.Resolution(item.Resolution), Required: item.Required, Secret: item.Secret, Default: item.Default, Aliases: item.Aliases}
}

// ResolveString reuses existing Matrix conversions. It deliberately does not run
// global rules: other nodes may have distinct explicit values or initialize later.
func (c *Catalog) ResolveString(r *matrixconfig.ConfigResolver, key, explicit string) (string, error) {
	item, found := c.Item(key)
	if !found {
		return "", problem(key, "unknown_key", "")
	}
	if item.Type != "string" && item.Type != "url" && item.Type != "path" && item.Type != "ref" && item.Type != "secret" {
		return "", problem(key, "value_type", "")
	}
	s := spec(item)
	if item.Secret {
		s.Resolution = matrixconfig.ResolutionPlaceholder
	} else if strings.TrimSpace(explicit) != "" {
		s.Resolution = matrixconfig.ResolutionNodeExplicit
		s.Explicit = explicit
	}
	value, _, err := matrixconfig.Resolve[string](r, s)
	if err != nil {
		return "", problem(key, "resolve", "")
	}
	return value, nil
}

// Resolve applies fixed source policy once, then validates the complete typed
// view once. Provider errors are never converted into "missing" or defaulted.
func (c *Catalog) Resolve(ctx context.Context, sources Sources, checks ...Check) (Resolved, Issues) {
	r := matrixconfig.NewConfigResolver(matrixconfig.WithValueSources(ctx, sources.Env, sources.Business))
	return c.resolve(ctx, r, sources.Explicit, checks...)
}

// ResolveWithResolver allows a host to validate once using the same resolver
// installed for its resource nodes. It supplies no module-global node override.
func (c *Catalog) ResolveWithResolver(r *matrixconfig.ConfigResolver, checks ...Check) (Resolved, Issues) {
	return c.resolve(context.Background(), r, nil, checks...)
}

func (c *Catalog) resolve(ctx context.Context, r *matrixconfig.ConfigResolver, explicit ValueSource, checks ...Check) (Resolved, Issues) {
	result := Resolved{values: map[string]any{}}
	var issues Issues
	for _, item := range c.items {
		s := spec(item)
		s.Required = false // completeness is checked on the full view below
		if item.Secret {
			s.Resolution = matrixconfig.ResolutionPlaceholder
		}
		if s.Resolution == matrixconfig.ResolutionNodeExplicit && explicit != nil {
			value, found, err := explicit.Lookup(ctx, item.Key)
			if err != nil {
				issues = append(issues, problem(item.Key, "source_read", "")...)
				continue
			}
			if found {
				s.Explicit = value
			}
		}
		raw, meta, err := matrixconfig.Resolve[any](r, s)
		if err != nil {
			issues = append(issues, problem(item.Key, "source_read", "")...)
			continue
		}
		if meta.Source == matrixconfig.SourceNone || raw == nil {
			continue
		}
		value, err := Convert(item, raw)
		if err != nil {
			issues = append(issues, problem(item.Key, "value_type", "")...)
			continue
		}
		result.values[item.Key] = value
	}
	issues = append(issues, c.Validate(result.values)...)
	for _, check := range checks {
		if check != nil {
			issues = append(issues, check(result)...)
		}
	}
	return result, issues
}

// Convert retains Matrix scalar/list conversion semantics. Wire serializers
// should keep the source value separately instead of round-tripping via JSON.
func Convert(item Item, raw any) (any, error) {
	if (item.Type == "int" || item.Type == "int64") && raw != nil {
		v := reflect.ValueOf(raw)
		switch v.Kind() {
		case reflect.Float32, reflect.Float64:
			n := v.Float()
			if math.IsNaN(n) || math.IsInf(n, 0) || math.Trunc(n) != n || math.Abs(n) > 9007199254740991 {
				return nil, problem(item.Key, "value_type", "")
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			// 用十进制字符串进入有范围检查的 ParseInt，禁止 uint64 静默回绕。
			raw = strconv.FormatUint(v.Uint(), 10)
		}
	}
	// JSON 元数据使用 Number 保存精度，转换器只接受普通 string。
	if number, ok := raw.(json.Number); ok {
		raw = number.String()
	}
	target := cnst.STRING
	switch item.Type {
	case "bool":
		target = cnst.BOOL
	case "int":
		target = cnst.INT
	case "int64":
		target = cnst.INT64
	case "float":
		target = cnst.FLOAT
	case "string_list":
		target = cnst.MType("[]string")
	}
	value, err := utils.Convert(raw, target)
	if err != nil {
		return nil, problem(item.Key, "value_type", "")
	}
	if _, err := json.Marshal(value); err != nil {
		return nil, problem(item.Key, "value_type", "")
	}
	return value, nil
}
