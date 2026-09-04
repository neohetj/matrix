package config

// WithValueTransform 在来源读取边界适配显式业务输入格式，随后仍由 Catalog 校验。
// 适配器不应重新读取来源或缓存配置；Reader Snapshot 会冻结适配后的值。
func WithValueTransform(transform func(key string, value any) (any, error)) ResolverOption {
	return func(r *ConfigResolver) { r.valueTransform = transform }
}
