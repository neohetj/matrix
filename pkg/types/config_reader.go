package types

import (
	"context"
	"errors"
)

// ConfigOverride 显式区分未提供值与 false、0 等合法节点值。
type ConfigOverride struct {
	Present bool
	Value   any
}

// ConfigReader 是模块配置的实例级读取契约，不公开配置值集合。
type ConfigReader interface {
	LookupConfig(context.Context, string, ConfigOverride) (any, bool, error)
}

// ConfigReaderAware 声明节点需要在 Init 前接收模块配置。
type ConfigReaderAware interface {
	SetConfigReader(ConfigReader)
}

// NodePoolAware 声明节点在 Init 前需要所属 Engine 的资源池，不允许隐式全局查找。
type NodePoolAware interface {
	SetNodePool(NodePool)
}

// ConfigReaderProvider 根据显式模块标识提供 Engine 内配置实例。
type ConfigReaderProvider interface {
	ConfigReader(moduleID string) (ConfigReader, bool)
}

// NodeConfigReaderProvider 根据宿主声明的节点归属提供配置，不猜测路径或前缀。
type NodeConfigReaderProvider interface {
	ConfigReaderForNode(nodeID string) (ConfigReader, bool)
}

// RuleChainConfigReaderProvider 按规则链归属提供运行时节点配置，避免同名局部节点串值。
type RuleChainConfigReaderProvider interface {
	ConfigReaderForRuleChain(chainID string) (ConfigReader, bool)
}

// RuleChainConfigImportValidator 在 flatten 后校验导入链没有跨越配置模块边界。
type RuleChainConfigImportValidator interface {
	ValidateRuleChainConfigImports(chainID string, imports []string) error
}

// ErrConfigReaderUnavailable 表示消费者没有获得明确的配置实例，错误不携带配置值。
var ErrConfigReaderUnavailable = errors.New("module configuration reader unavailable")

// ErrConfigInitialization 表示配置消费者初始化失败，避免传播可能含凭据的底层 cause。
var ErrConfigInitialization = errors.New("module configuration consumer initialization failed")
