package catalog

import (
	"context"
	"github.com/neohetj/matrix/pkg/config"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestReaderValidateProvided 验证启动检查使用同一来源快照，拒绝错误而不提前要求未启用功能的缺失项。
func TestReaderValidateProvided(t *testing.T) {
	ctx := context.Background()
	definition, err := Load(ctx, Documents{{Name: "runtime_catalog.yaml", Content: []byte(`version: "2"
module: sample
domain: runtime
items:
  - {key: COUNT, owner: sample, type: int, description: Count, resolution: placeholder, default: 5}
  - {key: TOKEN, owner: sample, type: secret, description: Token, resolution: placeholder, secret: true, required: true, schema: {pattern: '\S'}}
`)}})
	require.NoError(t, err)
	for _, tc := range []struct {
		name    string
		values  Values
		invalid bool
	}{
		{"missing required deferred", Values{}, false},
		{"zero present", Values{"COUNT": 0}, false},
		{"invalid integer", Values{"COUNT": "synthetic-invalid"}, true},
		{"blank secret", Values{"TOKEN": "   "}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader, err := NewReader(definition, config.NewConfigResolver(config.WithValueSources(ctx, tc.values, Values{})))
			require.NoError(t, err)
			tc.values["COUNT"] = 8
			err = reader.ValidateProvided(ctx)
			if tc.invalid {
				require.Error(t, err)
				require.NotContains(t, err.Error(), "synthetic-invalid")
			} else {
				require.NoError(t, err)
			}
		})
	}
}
