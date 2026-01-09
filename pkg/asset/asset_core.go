package asset

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sync"

	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
)

var (
	AssetUnmarshalFailed     = &types.Fault{Code: cnst.CodeAssetUnmarshalFailed, Message: "failed to unmarshal asset URI"}
	AssetInvalidURI          = &types.Fault{Code: cnst.CodeAssetInvalidURI, Message: "invalid uri"}
	AssetEmptyURI            = &types.Fault{Code: cnst.CodeAssetEmptyURI, Message: "empty uri"}
	AssetSchemeNotRegistered = &types.Fault{Code: cnst.CodeAssetSchemeNotRegistered, Message: "no handler registered for scheme"}
	AssetTypeMismatch        = &types.Fault{Code: cnst.CodeAssetTypeMismatch, Message: "type mismatch"}
	AssetCannotSetForNonURI  = &types.Fault{Code: cnst.CodeAssetCannotSetForNonURI, Message: "cannot set value for non-URI asset"}
)

// ---------------- Asset Context ------------------

// Option 定义了修改 AssetContext 的函数签名
type Option func(*AssetContext)

// AssetContext 聚合了解析 URI 所需的运行时上下文。
// 它是线程安全的，支持通过 Option 模式进行扩展。
type AssetContext struct {
	// 基础核心上下文
	ruleMsg types.RuleMsg
	nodeCtx types.NodeCtx
	config  types.ConfigMap

	// 扩展存储，用于存储额外的信息 (如 TraceInfo, UserSession 等)
	extras map[any]any
	mu     sync.RWMutex
}

// NewAssetContext 创建一个新的上下文实例
func NewAssetContext(opts ...Option) *AssetContext {
	rc := &AssetContext{
		extras: make(map[any]any),
	}
	for _, opt := range opts {
		opt(rc)
	}
	return rc
}

// 基础 Option 实现
func WithRuleMsg(msg types.RuleMsg) Option {
	return func(rc *AssetContext) {
		rc.ruleMsg = msg
	}
}

func WithNodeCtx(ctx types.NodeCtx) Option {
	return func(rc *AssetContext) {
		rc.nodeCtx = ctx
	}
}

func WithConfig(cfg types.ConfigMap) Option {
	return func(rc *AssetContext) {
		rc.config = cfg
	}
}

// WithValue 允许像 context.WithValue 一样存储任意键值对
func WithValue(key, val any) Option {
	return func(rc *AssetContext) {
		rc.mu.Lock()
		defer rc.mu.Unlock()
		rc.extras[key] = val
	}
}

// WithNodePool provides the NodePool to the context
func WithNodePool(pool types.NodePool) Option {
	return WithValue("node_pool", pool)
}

// GetNodePool retrieves the NodePool from the context
func GetNodePool(rc *AssetContext) types.NodePool {
	val := rc.Value("node_pool")
	if pool, ok := val.(types.NodePool); ok {
		return pool
	}
	return nil
}

// Getter 方法

func (rc *AssetContext) RuleMsg() types.RuleMsg {
	return rc.ruleMsg
}

func (rc *AssetContext) NodeCtx() types.NodeCtx {
	return rc.nodeCtx
}

func (rc *AssetContext) Config() types.ConfigMap {
	return rc.config
}

// Value 获取扩展上下文中的值
func (rc *AssetContext) Value(key any) any {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.extras[key]
}

// ---------------- Asset ------------------

// Asset[T] 是一个泛型的资源引用，代表一个将在运行时解析为 T 类型的资源。
type Asset[T any] struct {
	URI string
}

// UnmarshalJSON implements the json.Unmarshaler interface for Asset[T].
// It allows parsing a JSON string directly into the URI field of the Asset.
func (a *Asset[T]) UnmarshalJSON(data []byte) error {
	var uri string
	if err := json.Unmarshal(data, &uri); err != nil {
		return AssetUnmarshalFailed.Wrap(err)
	}
	a.URI = uri
	return nil
}

// MarshalJSON implements the json.Marshaler interface for Asset[T].
// It serializes the Asset's URI as a JSON string.
func (a Asset[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.URI)
}

// UnmarshalText implements the encoding.TextUnmarshaler interface for Asset[T].
// This allows mapstructure to decode a string directly into the Asset struct.
func (a *Asset[T]) UnmarshalText(text []byte) error {
	a.URI = string(text)
	return nil
}

// Resolve 在给定的上下文中解析资源
func (a Asset[T]) Resolve(ctx *AssetContext) (T, error) {
	var zero T
	if a.URI == "" {
		return zero, AssetInvalidURI.Wrap(fmt.Errorf("uri is empty"))
	}

	// 1. 尝试解析 URI 字符串
	// 如果包含换行符等控制字符，url.Parse 会报错。
	// 对于 Prompt 模板等场景，这通常意味着它是一个纯文本。
	u, err := url.Parse(a.URI)
	if err != nil {
		return zero, AssetInvalidURI.Wrap(err)
	}

	// 2. 查找 Handler
	h := GetHandler(u.Scheme)
	if h == nil {
		// 如果没有找到对应的 Scheme 处理程序，直接报错
		return zero, AssetSchemeNotRegistered.Wrap(fmt.Errorf(u.Scheme))
	}

	if validator, ok := h.(ValidatingHandler); ok {
		if err := validator.Validate(u); err != nil {
			return zero, err
		}
	}

	// 3. 委托解析
	val, err := h.Handle(u, ctx)
	if err != nil {
		return zero, err
	}

	// 4. 类型转换
	if tVal, ok := val.(T); ok {
		return tVal, nil
	}
	if tVal, ok := coerceBasicPointer[T](val); ok {
		return tVal, nil
	}

	return zero, AssetTypeMismatch.Wrap(fmt.Errorf("expected %T, got %T", zero, val))
}

func coerceBasicPointer[T any](val any) (T, bool) {
	var zero T
	switch any(zero).(type) {
	case string:
		if v, ok := val.(*string); ok && v != nil {
			return any(*v).(T), true
		}
	case int64:
		if v, ok := val.(*int64); ok && v != nil {
			return any(*v).(T), true
		}
	case float64:
		if v, ok := val.(*float64); ok && v != nil {
			return any(*v).(T), true
		}
	case bool:
		if v, ok := val.(*bool); ok && v != nil {
			return any(*v).(T), true
		}
	case map[string]any:
		if v, ok := val.(*map[string]any); ok && v != nil {
			return any(*v).(T), true
		}
	case map[string]string:
		if v, ok := val.(*map[string]string); ok && v != nil {
			return any(*v).(T), true
		}
	}
	return zero, false
}

// Set 在给定的上下文中设置资源值
func (a Asset[T]) Set(ctx *AssetContext, val T) error {
	if a.URI == "" {
		return AssetEmptyURI
	}

	// 1. 解析 URI 字符串
	u, err := url.Parse(a.URI)
	if err != nil {
		return AssetInvalidURI.Wrap(err)
	}

	if u.Scheme == "" {
		return AssetCannotSetForNonURI.Wrap(fmt.Errorf(a.URI))
	}

	// 2. 查找 Handler
	h := GetHandler(u.Scheme)
	if h == nil {
		return AssetSchemeNotRegistered.Wrap(fmt.Errorf(u.Scheme))
	}

	if validator, ok := h.(ValidatingHandler); ok {
		if err := validator.Validate(u); err != nil {
			return err
		}
	}

	// 3. 委托设置
	return h.Set(u, ctx, val)
}

// ---------------- Scheme Register ------------------

var (
	registryMu sync.RWMutex
	registry   = make(map[string]SchemeHandler)
)

// SchemeHandler 定义了处理特定 URI Scheme 的接口
type SchemeHandler interface {
	// Handle 解析 URI 并返回对应的资源值
	// uri: 完整的解析后的 URL 对象
	Handle(uri *url.URL, ctx *AssetContext) (any, error)

	// Set 设置 URI 对应的资源值
	Set(uri *url.URL, ctx *AssetContext, value any) error

	// NormalizeAssetURI provides URI normalization.
	NormalizeAssetURI(uri string) string
}

// ValidatingHandler provides URI validation before handling.
type ValidatingHandler interface {
	Validate(uri *url.URL) error
}

func RegisterScheme(scheme string, handler SchemeHandler) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[scheme] = handler
}

func GetHandler(scheme string) SchemeHandler {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return registry[scheme]
}

// InitRegistry initializes the global registry with default handlers.
func InitRegistry() {
	RegisterScheme(cnst.SchemeRuleMsg, RuleMsgAsset{})
	RegisterScheme(cnst.SchemeConfig, ConfigAsset{})
	RegisterScheme(cnst.SchemeRel, RelAsset{})
	RegisterScheme(cnst.SchemeRef, RefAsset{})
}

func init() {
	InitRegistry()
}
