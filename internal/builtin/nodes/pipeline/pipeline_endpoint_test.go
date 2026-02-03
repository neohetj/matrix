package pipeline

import (
	"context"
	"testing"
	"time"

	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/test/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPipelineEndpointNode_Init(t *testing.T) {
	node := &PipelineEndpointNode{}
	node.BaseNode = *types.NewBaseNode(PipelineEndpointNodeType, types.NodeMetadata{})

	// Valid Config
	config := types.ConfigMap{
		"stages": []any{
			map[string]any{
				"name":        "Stage1",
				"id":          "s1",
				"concurrency": 2,
				"processor":   map[string]interface{}{"id": "chain1", "type": "chain"},
			},
		},
		"exposedChannels": map[string]any{
			"input": "s1_in",
		},
		"channelManager": "ref://shared-cm",
	}

	err := node.Init(config)
	assert.NoError(t, err)
	assert.NotNil(t, node.activeChannels)
	assert.Equal(t, 1, len(node.config.Stages))
	assert.Equal(t, "Stage1", node.config.Stages[0].Name)
}

func TestPipelineEndpointNode_StartAndProcess(t *testing.T) {
	node := &PipelineEndpointNode{}
	node.BaseNode = *types.NewBaseNode(PipelineEndpointNodeType, types.NodeMetadata{})
	node.SetID("test-pipeline-node")

	// Setup Shared Channel Manager
	cmDSL := `
	{
		"metadata": {
			"nodes": [
				{
					"id": "shared-cm",
					"type": "resource/channel_manager",
					"name": "Shared CM"
				}
			]
		}
	}
	`
	pool := registry.Default.GetSharedNodePool()
	_, err := pool.Load([]byte(cmDSL), registry.Default.GetNodeManager())
	assert.NoError(t, err)

	config := types.ConfigMap{
		"stages": []any{
			map[string]any{
				"name":         "Stage1",
				"id":           "s1",
				"concurrency":  1,
				"processor":    map[string]interface{}{"id": "mock-chain", "type": "chain"},
				"inputChannel": "s1_in",
			},
		},
		"exposedChannels": map[string]any{
			"input": "s1_in",
		},
		"channelManager": "ref://shared-cm",
	}
	node.Init(config)

	// Mock Runtime
	mockRuntime := new(utils.MockRuntime)
	mockRuntime.On("Execute", mock.Anything, "", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		msg := args.Get(2).(types.RuleMsg)
		onEnd := args.Get(3).(func(types.RuleMsg, error))
		onEnd(msg, nil)
	})

	// Mock RuntimePool
	mockPool := new(MockRuntimePoolForPipeline)
	mockPool.On("Get", "mock-chain").Return(mockRuntime, true)

	node.SetRuntimePool(mockPool)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = node.Start(ctx)
	assert.NoError(t, err)

	// Verify channels created
	instance, err := pool.GetInstance("shared-cm")
	assert.NoError(t, err)
	cm := instance.(*ChannelManager)

	ch, err := cm.Get(node.ID(), "s1_in")
	if assert.NoError(t, err) && assert.NotNil(t, ch) {
		// Push data
		ch <- types.NewMsg("test", "", nil, types.NewDataT())
	}

	// Wait for processing (async)
	time.Sleep(100 * time.Millisecond)

	mockRuntime.AssertExpectations(t)
	node.Stop()
}
