package ops

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/neohet/matrix/pkg/types"
)

func TestMachineNode_New(t *testing.T) {
	node := machineNodePrototype.New()
	assert.NotNil(t, node)
	_, ok := node.(*MachineNode)
	assert.True(t, ok)
}

func TestMachineNode_Init(t *testing.T) {
	tests := []struct {
		name        string
		config      types.Config
		expectErr   bool
		expectedCfg MachineNodeConfiguration
	}{
		{
			name: "Valid config",
			config: types.Config{
				"address":              "127.0.0.1",
				"credentialSecretName": "ssh-key",
			},
			expectErr: false,
			expectedCfg: MachineNodeConfiguration{
				Address:              "127.0.0.1",
				CredentialSecretName: "ssh-key",
			},
		},
		{
			name:      "Empty config",
			config:    types.Config{},
			expectErr: false, // Address is not validated at Init stage
			expectedCfg: MachineNodeConfiguration{
				Address: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := machineNodePrototype.New().(*MachineNode)
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
