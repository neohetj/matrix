package catalog

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/neohetj/matrix/pkg/types"
	"github.com/stretchr/testify/require"
)

// TestAuditBusinessSecrets 检查主键与别名，只报告键名，不读取普通配置或输出凭据。
func TestAuditBusinessSecrets(t *testing.T) {
	definition, err := Load(context.Background(), Documents{{Name: "sample_catalog.yaml", Content: []byte(`version: "1"
module: sample
domain: config
items:
  - {key: TOKEN, owner: sample, type: secret, secret: true, description: token, resolution: placeholder, aliases: [OLD_TOKEN, EMPTY_TOKEN]}
  - {key: ZERO, owner: sample, type: secret, secret: true, description: zero, resolution: placeholder}
  - {key: SWITCH, owner: sample, type: secret, secret: true, description: switch, resolution: placeholder}
  - {key: ORDINARY, owner: sample, type: string, description: ordinary, resolution: placeholder}
`)}})
	require.NoError(t, err)
	findings := AuditBusinessSecrets(definition, types.ConfigMap{"TOKEN": "credential", "OLD_TOKEN": "old-credential", "EMPTY_TOKEN": " \t", "ZERO": 0, "SWITCH": false, "ORDINARY": "ordinary"})
	require.Equal(t, []SecretFinding{{Key: "SWITCH"}, {Key: "TOKEN"}, {Key: "TOKEN", Alias: "OLD_TOKEN"}, {Key: "ZERO"}}, findings)
	encoded, err := json.Marshal(findings)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "credential")
	require.Empty(t, AuditBusinessSecrets(nil, types.ConfigMap{"TOKEN": "secret"}))
	require.Empty(t, AuditBusinessSecrets(definition, nil))
}

// TestLegacyRefCatalog 验证旧资源引用类型保留为字符串而不主动解析资源。
func TestLegacyRefCatalog(t *testing.T) {
	definition, err := Load(context.Background(), Documents{{Name: "sample_catalog.yaml", Content: []byte(`version: "1"
module: sample
domain: config
items:
  - {key: STORE_REF, owner: sample, type: ref, description: store, resolution: placeholder, default: "ref://shared/store"}
`)}})
	require.NoError(t, err)
	resolved, issues := definition.Resolve(context.Background(), Sources{})
	require.Empty(t, issues)
	value, found := resolved.Lookup("STORE_REF")
	require.True(t, found)
	require.Equal(t, "ref://shared/store", value)
}
