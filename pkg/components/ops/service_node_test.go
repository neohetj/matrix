package ops

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/neohet/matrix/pkg/types"
)

func TestServiceNode_New(t *testing.T) {
	node := serviceNodePrototype.New()
	assert.NotNil(t, node)
	_, ok := node.(*ServiceNode)
	assert.True(t, ok)
}

func TestServiceNode_Init(t *testing.T) {
	tests := []struct {
		name        string
		config      types.Config
		expectErr   bool
		expectedCfg ServiceNodeConfiguration
	}{
		{
			name: "Valid full config",
			config: types.Config{
				"ruleChainRefs": []string{"chain1", "chain2"},
				"endpoints": []interface{}{
					map[string]interface{}{
						"endpointRef": "ep1",
						"type":        "http",
						"path":        "/api/v1",
						"method":      "POST",
					},
				},
			},
			expectErr: false,
			expectedCfg: ServiceNodeConfiguration{
				RuleChainRefs: []string{"chain1", "chain2"},
				Endpoints: []ServiceEndpoint{
					{
						EndpointRef: "ep1",
						Type:        "http",
						Path:        "/api/v1",
						Method:      "POST",
					},
				},
			},
		},
		{
			name:      "Empty config",
			config:    types.Config{},
			expectErr: false,
			expectedCfg: ServiceNodeConfiguration{
				RuleChainRefs: nil,
				Endpoints:     nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := serviceNodePrototype.New().(*ServiceNode)
			err := node.Init(tt.config)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCfg, node.nodeConfig)
			}
		})
	}
}
