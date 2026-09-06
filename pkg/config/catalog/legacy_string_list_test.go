package catalog

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLegacyStringListTextDefaultPreservesWireFormat 校验 v1 文本列表沿用模块既有字符串协议。
func TestLegacyStringListTextDefaultPreservesWireFormat(t *testing.T) {
	raw := []byte(`version: "1"
module: brokerx
domain: connector_oauth
items:
  - key: BROKERX_CONNECTOR_GITHUB_ALLOWED_SCOPES
    owner: brokerx
    type: string_list
    description: Allowed GitHub OAuth scopes.
    resolution: placeholder
    default: "read:user repo read:org"
    secret: false
`)

	c, err := Load(context.Background(), Documents{{Name: "connector_oauth_catalog.yaml", Content: raw}})
	require.NoError(t, err)
	item, ok := c.Item("BROKERX_CONNECTOR_GITHUB_ALLOWED_SCOPES")
	require.True(t, ok)
	require.Equal(t, "string", item.Type)
	require.Equal(t, "read:user repo read:org", item.Default)
	require.Empty(t, c.ValidateProvided(map[string]any{
		"BROKERX_CONNECTOR_GITHUB_ALLOWED_SCOPES": "read:user repo read:org",
	}))
	_, err = Restore(c.Freeze())
	require.NoError(t, err)
}

// TestV2StringListTextDefaultRejected 保持新版 Catalog 的数组类型约束。
func TestV2StringListTextDefaultRejected(t *testing.T) {
	raw := []byte(`version: "2"
module: brokerx
domain: connector_oauth
items:
  - key: SCOPES
    owner: brokerx
    type: string_list
    description: Allowed GitHub OAuth scopes.
    resolution: placeholder
    default: "read:user repo read:org"
    secret: false
`)

	_, err := Load(context.Background(), Documents{{Name: "connector_oauth_catalog.yaml", Content: raw}})
	require.ErrorContains(t, err, "value_type: SCOPES")
}
