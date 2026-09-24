package catalog

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/neohetj/matrix/pkg/config"
	"github.com/neohetj/matrix/pkg/types"
)

// Reader 持有一个模块实例的不可变来源，所有消费者共享规则但不共享全局状态。
type Reader struct {
	definition *Catalog
	resolver   *config.ConfigResolver
	items      map[string]Item
}

// NewReader 冻结声明和来源，不隐式执行整模块条件校验。
func NewReader(definition *Catalog, resolver *config.ConfigResolver) (*Reader, error) {
	if definition == nil || resolver == nil {
		return nil, problem("", "reader_missing", "")
	}
	items := make(map[string]Item, len(definition.items))
	specs := make([]config.ConfigSpec, 0, len(definition.items))
	for _, item := range definition.Items() {
		items[item.Key] = item
		specs = append(specs, spec(item))
	}
	snapshot, err := resolver.Snapshot(specs)
	if err != nil {
		return nil, problem("", "source_read", "")
	}
	return &Reader{definition: definition, resolver: snapshot, items: items}, nil
}

// LookupConfig 复用来源核心和字段转换；节点 override 不污染全局值视图。
func (r *Reader) LookupConfig(ctx context.Context, key string, override types.ConfigOverride) (any, bool, error) {
	return r.lookupConfig(ctx, key, override, true)
}

// lookupConfig 共用字段转换，仅允许启动预检延后缺失项的完整性检查。
func (r *Reader) lookupConfig(ctx context.Context, key string, override types.ConfigOverride, requireMissing bool) (any, bool, error) {
	if r == nil || r.definition == nil || ctx == nil {
		return nil, false, problem(key, "reader_missing", "")
	}
	if ctx.Err() != nil {
		return nil, false, problem(key, "context_canceled", "")
	}
	item, found := r.items[key]
	if !found {
		return nil, false, problem(key, "unknown_key", "")
	}
	s := spec(item)
	if item.Secret {
		if override.Present {
			return nil, false, problem(key, "secret_override", "")
		}
		s.Resolution = config.ResolutionPlaceholder
	} else if override.Present {
		s.Resolution, s.Explicit = config.ResolutionNodeExplicit, override.Value
	}
	s.Required = false
	raw, meta, err := config.Resolve[any](r.resolver, s)
	if err != nil {
		return nil, false, problem(key, "source_read", "")
	}
	if meta.Source == config.SourceNone || raw == nil {
		if item.Required && requireMissing {
			return nil, false, problem(key, "required", "")
		}
		return nil, false, nil
	}
	value, err := Convert(item, raw)
	if err != nil {
		return nil, false, problem(key, "value_type", "")
	}
	if r.definition.fields[key].Validate(clone(value)) != nil {
		return nil, false, problem(key, "schema", "")
	}
	return copyValue(value), true, nil
}

// Read 读取 Catalog 声明类型并区分缺失与错误。
func Read[T any](ctx context.Context, reader types.ConfigReader, key string) (T, bool, error) {
	return ReadNode[T](ctx, reader, key, types.ConfigOverride{})
}

// ReadNode 使用显式 presence 支持 false/0，禁止错误后返回未校验值。
func ReadNode[T any](ctx context.Context, reader types.ConfigReader, key string, override types.ConfigOverride) (T, bool, error) {
	var zero T
	if reader == nil {
		return zero, false, problem(key, "reader_missing", "")
	}
	raw, found, err := reader.LookupConfig(ctx, key, override)
	if err != nil || !found {
		return zero, found, err
	}
	if value, ok := raw.(T); ok {
		return value, true, nil
	}
	// int 与 int64 只作有界整数适配，绝不经过 float64。
	switch any(zero).(type) {
	case int64:
		if value, ok := raw.(int); ok {
			return any(int64(value)).(T), true, nil
		}
	case int:
		if value, ok := raw.(int64); ok && int64(int(value)) == value {
			return any(int(value)).(T), true, nil
		}
	}
	return zero, false, problem(key, "target_type", "")
}

// ReadDuration 读取明确单位的时长，整数字符串不会按 key 名猜测单位。
func ReadDuration(ctx context.Context, reader types.ConfigReader, key string, bareUnit time.Duration) (time.Duration, bool, error) {
	if reader == nil {
		return 0, false, problem(key, "reader_missing", "")
	}
	raw, found, err := reader.LookupConfig(ctx, key, types.ConfigOverride{})
	if err != nil || !found {
		return 0, found, err
	}
	var text string
	switch value := raw.(type) {
	case string:
		text = value
	case int:
		text = strconv.FormatInt(int64(value), 10)
	case int64:
		text = strconv.FormatInt(value, 10)
	default:
		return 0, false, problem(key, "duration", "")
	}
	value, err := ParseDuration(text, bareUnit)
	if err != nil {
		return 0, false, problem(key, "duration", "")
	}
	return value, true, nil
}

// ParseDuration 接受 Go 时长或指定单位的十进制整数，检测乘法溢出。
func ParseDuration(raw string, bareUnit time.Duration) (time.Duration, error) {
	if value, err := time.ParseDuration(raw); err == nil {
		return value, nil
	}
	if bareUnit <= 0 {
		return 0, problem("", "duration_unit", "")
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value > math.MaxInt64/int64(bareUnit) || value < math.MinInt64/int64(bareUnit) {
		return 0, problem("", "duration", "")
	}
	return time.Duration(value) * bareUnit, nil
}

// String 禁止日志格式化泄露来源值。
func (r *Reader) String() string { return "CatalogReader{values:redacted}" }

// GoString 同样保护 %#v 调试输出。
func (r *Reader) GoString() string { return r.String() }

// MarshalJSON 只公开对象种类，不公开配置数据。
func (r *Reader) MarshalJSON() ([]byte, error) { return []byte(`{"kind":"catalog_reader"}`), nil }

// Format 对所有 fmt verb 输出安全描述，不遍历私有值。
func (r *Reader) Format(state fmt.State, verb rune) { _, _ = state.Write([]byte(r.String())) }
