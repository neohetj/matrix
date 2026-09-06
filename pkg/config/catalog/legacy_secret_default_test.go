package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/neohetj/matrix/pkg/config"
	"github.com/stretchr/testify/require"
)

// TestLegacySecretEmptyDefault 校验旧目录空占位兼容不产生凭据或冻结默认值。
func TestLegacySecretEmptyDefault(t *testing.T) {
	for _, value := range []string{"", "{}", "[]"} {
		t.Run(fmt.Sprintf("%q", value), func(t *testing.T) {
			raw := fmt.Sprintf("version: '1'\nmodule: sample\ndomain: runtime\nitems:\n  - {key: KEYRING, owner: sample, type: string, description: keyring, resolution: placeholder, secret: true, default: %q}\n", value)
			c, err := Load(context.Background(), Documents{{Name: "runtime_catalog.yaml", Content: []byte(raw)}})
			require.NoError(t, err)
			item, ok := c.Item("KEYRING")
			require.True(t, ok)
			require.Nil(t, item.Default)
			reader, err := NewReader(c, config.NewConfigResolver(config.WithValueSources(context.Background(), Values{}, Values{})))
			require.NoError(t, err)
			_, found, err := Read[string](context.Background(), reader, "KEYRING")
			require.NoError(t, err)
			require.False(t, found)
			encoded, err := json.Marshal(c.Freeze())
			require.NoError(t, err)
			require.NotContains(t, string(encoded), `"default"`)
			_, err = Restore(c.Freeze())
			require.NoError(t, err)
		})
	}
}

// TestSecretDefaultCompatibilityBoundary 保持新目录和非空旧凭据默认值的拒绝规则。
func TestSecretDefaultCompatibilityBoundary(t *testing.T) {
	for _, tc := range []struct{ version, value string }{{"1", "synthetic-secret"}, {"1", `{"old":"synthetic-secret"}`}, {"2", "{}"}, {"2", ""}} {
		raw := fmt.Sprintf("version: %q\nmodule: sample\ndomain: runtime\nitems:\n  - {key: KEYRING, owner: sample, type: string, description: keyring, resolution: placeholder, secret: true, default: %q}\n", tc.version, tc.value)
		_, err := Load(context.Background(), Documents{{Name: "runtime_catalog.yaml", Content: []byte(raw)}})
		require.Error(t, err)
		require.NotContains(t, err.Error(), "synthetic-secret")
	}
}
