package catalog

import (
	"context"
	"errors"

	"github.com/neohetj/matrix/pkg/types"
)

// ValidateProvided 校验当前来源快照的全部已提供值与默认值；缺失必填和关联条件留给能力初始化或完整 Validate。
// 它不重新读取环境，不输出配置值，也不把解析错误当成缺失。
func (r *Reader) ValidateProvided(ctx context.Context) error {
	if r == nil || r.definition == nil {
		return problem("", "reader_missing", "")
	}
	var failures []error
	for _, item := range r.definition.items {
		if _, _, err := r.lookupConfig(ctx, item.Key, types.ConfigOverride{}, false); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}
