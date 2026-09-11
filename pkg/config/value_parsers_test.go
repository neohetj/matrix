package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestParseByteSize 保留常用二进制后缀并拒绝无效或溢出的容量。
func TestParseByteSize(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want int64
	}{{"0", 0}, {"5", 5}, {" 2KB ", 2048}, {"3m", 3 << 20}, {"1GB", 1 << 30}} {
		value, err := ParseByteSize(tc.raw)
		require.NoError(t, err)
		require.Equal(t, tc.want, value)
	}
	for _, raw := range []string{"-1", "1.5MB", "9223372036854775807G", "secret-invalid"} {
		_, err := ParseByteSize(raw)
		require.Error(t, err)
		require.NotContains(t, err.Error(), raw)
	}
}

// TestParseBool 验证显式宽松语法，不改变默认 Catalog 类型转换的语义。
func TestParseBool(t *testing.T) {
	for _, raw := range []string{"true", "YES", " on ", "enabled", "1", "y"} {
		value, err := ParseBool(raw)
		require.NoError(t, err)
		require.True(t, value)
	}
	for _, raw := range []string{"false", "NO", "off", "disabled", "0", "n"} {
		value, err := ParseBool(raw)
		require.NoError(t, err)
		require.False(t, value)
	}
	_, err := ParseBool("secret-invalid")
	require.Error(t, err)
	require.NotContains(t, err.Error(), "secret-invalid")
}

// TestParseCSV 验证逗号列表规范化保留顺序和重复值。
func TestParseCSV(t *testing.T) {
	require.Equal(t, []string{"a", "b", "a"}, ParseCSV(" a, , b,a ,"))
	require.Empty(t, ParseCSV(" , "))
}
