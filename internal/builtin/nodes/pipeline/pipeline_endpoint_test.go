package pipeline

import (
	"context"
	"errors"
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

func TestPipelineEndpointNode_ProcessData_RuntimeErrorBlocksOutput(t *testing.T) {
	node := &PipelineEndpointNode{}
	node.BaseNode = *types.NewBaseNode(PipelineEndpointNodeType, types.NodeMetadata{})
	node.SetID("test-pipeline-node")
	node.activeChannels = map[string]chan types.RuleMsg{
		"out": make(chan types.RuleMsg, 1),
	}

	mockRuntime := new(utils.MockRuntime)
	mockRuntime.On("Execute", mock.Anything, "", mock.Anything, mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			onEnd := args.Get(3).(func(types.RuleMsg, error))
			onEnd(nil, errors.New("stage failed"))
		})

	mockPool := new(MockRuntimePoolForPipeline)
	mockPool.On("Get", "mock-chain").Return(mockRuntime, true)
	node.SetRuntimePool(mockPool)

	stage := PipelineStageConfig{
		Name:          "Stage1",
		Processor:     ProcessorConfig{ID: "mock-chain", Type: "chain"},
		OutputChannel: "out",
	}
	inMsg := types.NewMsg("input", "", nil, types.NewDataT())

	node.processData(context.Background(), stage, inMsg)

	assert.Equal(t, 0, len(node.activeChannels["out"]))
	mockRuntime.AssertExpectations(t)
	mockPool.AssertExpectations(t)
}

func TestPipelineEndpointNode_ProcessData_MetadataErrorBlocksOutput(t *testing.T) {
	node := &PipelineEndpointNode{}
	node.BaseNode = *types.NewBaseNode(PipelineEndpointNodeType, types.NodeMetadata{})
	node.SetID("test-pipeline-node")
	node.activeChannels = map[string]chan types.RuleMsg{
		"out": make(chan types.RuleMsg, 1),
	}

	mockRuntime := new(utils.MockRuntime)
	mockRuntime.On("Execute", mock.Anything, "", mock.Anything, mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			onEnd := args.Get(3).(func(types.RuleMsg, error))
			result := types.NewMsg("result", "", types.Metadata{types.MetaError: "failed in metadata"}, types.NewDataT())
			onEnd(result, nil)
		})

	mockPool := new(MockRuntimePoolForPipeline)
	mockPool.On("Get", "mock-chain").Return(mockRuntime, true)
	node.SetRuntimePool(mockPool)

	stage := PipelineStageConfig{
		Name:          "Stage1",
		Processor:     ProcessorConfig{ID: "mock-chain", Type: "chain"},
		OutputChannel: "out",
	}
	inMsg := types.NewMsg("input", "", nil, types.NewDataT())

	node.processData(context.Background(), stage, inMsg)

	assert.Equal(t, 0, len(node.activeChannels["out"]))
	mockRuntime.AssertExpectations(t)
	mockPool.AssertExpectations(t)
}

func TestPipelineEndpointNode_ProcessData_ProjectsToStageRequiredInputs(t *testing.T) {
	node := &PipelineEndpointNode{}
	node.BaseNode = *types.NewBaseNode(PipelineEndpointNodeType, types.NodeMetadata{})
	node.SetID("test-pipeline-node")

	targetRuntime := &testProjectionRuntime{
		projection: types.RuleChainCoreObjAnalysis{
			RequiredInputs: types.CoreObjSet{ObjIDs: []string{"image_push_items"}},
		},
		executeFn: func(_ context.Context, _ string, msg types.RuleMsg, onEnd func(types.RuleMsg, error)) error {
			_, hasImagePushItems := msg.DataT().Get("image_push_items")
			_, hasScrapedPosts := msg.DataT().Get("ttscrapedposts")
			assert.True(t, hasImagePushItems)
			assert.False(t, hasScrapedPosts)
			onEnd(msg, nil)
			return nil
		},
	}

	mockPool := new(MockRuntimePoolForPipeline)
	mockPool.On("Get", "mock-chain").Return(targetRuntime, true)
	node.SetRuntimePool(mockPool)

	stage := PipelineStageConfig{
		Name:      "Stage1",
		Processor: ProcessorConfig{ID: "mock-chain", Type: "chain"},
	}
	inDataT := types.NewDataT()
	inDataT.Set("image_push_items", &testCoreObj{key: "image_push_items", body: "keep"})
	inDataT.Set("ttscrapedposts", &testCoreObj{key: "ttscrapedposts", body: "drop"})
	inMsg := types.NewMsg("input", "", nil, inDataT)

	node.processData(context.Background(), stage, inMsg)

	mockPool.AssertExpectations(t)
}

func TestPipelineEndpointNode_ProcessData_ProjectsToUnionOfDownstreamStageInputs(t *testing.T) {
	node := &PipelineEndpointNode{}
	node.BaseNode = *types.NewBaseNode(PipelineEndpointNodeType, types.NodeMetadata{})
	node.SetID("test-pipeline-node")

	stageOneRuntime := &testProjectionRuntime{
		projection: types.RuleChainCoreObjAnalysis{
			RequiredInputs: types.CoreObjSet{ObjIDs: []string{"stage1_only"}},
		},
		executeFn: func(_ context.Context, _ string, msg types.RuleMsg, onEnd func(types.RuleMsg, error)) error {
			_, hasStage1 := msg.DataT().Get("stage1_only")
			_, hasStage2 := msg.DataT().Get("stage2_only")
			_, hasStage3 := msg.DataT().Get("stage3_only")
			_, hasDrop := msg.DataT().Get("drop_me")
			assert.True(t, hasStage1)
			assert.True(t, hasStage2)
			assert.True(t, hasStage3)
			assert.False(t, hasDrop)
			onEnd(msg, nil)
			return nil
		},
	}
	stageTwoRuntime := &testProjectionRuntime{
		projection: types.RuleChainCoreObjAnalysis{
			RequiredInputs: types.CoreObjSet{ObjIDs: []string{"stage2_only"}},
		},
	}
	stageThreeRuntime := &testProjectionRuntime{
		projection: types.RuleChainCoreObjAnalysis{
			RequiredInputs: types.CoreObjSet{ObjIDs: []string{"stage3_only"}},
		},
	}

	mockPool := new(MockRuntimePoolForPipeline)
	mockPool.On("Get", "stage-1-chain").Return(stageOneRuntime, true)
	mockPool.On("Get", "stage-2-chain").Return(stageTwoRuntime, true)
	mockPool.On("Get", "stage-3-chain").Return(stageThreeRuntime, true)
	node.SetRuntimePool(mockPool)
	node.activeChannels = map[string]chan types.RuleMsg{
		"stage2_in": make(chan types.RuleMsg, 1),
		"stage3_in": make(chan types.RuleMsg, 1),
	}
	node.config = PipelineConfig{
		Stages: []PipelineStageConfig{
			{
				Name:          "Stage1",
				Processor:     ProcessorConfig{ID: "stage-1-chain", Type: "chain"},
				OutputChannel: "stage2_in",
			},
			{
				Name:          "Stage2",
				Processor:     ProcessorConfig{ID: "stage-2-chain", Type: "chain"},
				InputChannel:  "stage2_in",
				OutputChannel: "stage3_in",
			},
			{
				Name:         "Stage3",
				Processor:    ProcessorConfig{ID: "stage-3-chain", Type: "chain"},
				InputChannel: "stage3_in",
			},
		},
	}

	inDataT := types.NewDataT()
	inDataT.Set("stage1_only", &testCoreObj{key: "stage1_only", body: "keep-stage1"})
	inDataT.Set("stage2_only", &testCoreObj{key: "stage2_only", body: "keep-stage2"})
	inDataT.Set("stage3_only", &testCoreObj{key: "stage3_only", body: "keep-stage3"})
	inDataT.Set("drop_me", &testCoreObj{key: "drop_me", body: "drop"})
	inMsg := types.NewMsg("input", "", nil, inDataT)

	node.processData(context.Background(), node.config.Stages[0], inMsg)

	mockPool.AssertExpectations(t)
}
