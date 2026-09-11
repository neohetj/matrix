package catalog

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProvidedValidationDoesNotRewriteCompleteRules(t *testing.T) {
	c := loadFixture(t, fixture)
	require.Empty(t, c.ValidateProvided(map[string]any{"BACKEND": "remote"}))
	require.NotEmpty(t, c.ValidateProvided(map[string]any{"BACKEND": "invalid"}))
	require.NotEmpty(t, c.Validate(map[string]any{"BACKEND": "remote"}))
	require.NotEmpty(t, c.ValidateProvided(map[string]any{"TYPO": "value"}))
	// 空输入等同尚未提供；草稿可保存，执行时仍检查完整的必填约束。
	require.Empty(t, c.ValidateProvided(map[string]any{"BACKEND": ""}))
}
