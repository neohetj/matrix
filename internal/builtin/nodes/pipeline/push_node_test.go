package pipeline

import (
	"testing"
	"time"

	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/test/utils"
	"github.com/stretchr/testify/assert"
)

func containsURI(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func TestChannelPushNode_Execute(t *testing.T) {
	node := &ChannelPushNode{}
	node.BaseNode = *types.NewBaseNode(ChannelPushNodeType, types.NodeMetadata{})

	pipelineID := "test_pipeline"
	channelName := "test_channel"

	// Setup Shared Channel Manager
	cmDSL := `{"metadata":{"nodes":[{"id":"shared-cm-push","type":"resource/channel_manager","name":"Shared CM Push"}]}}`
	pool := registry.Default.GetSharedNodePool()
	pool.Load([]byte(cmDSL), registry.Default.GetNodeManager())
	inst, _ := pool.GetInstance("shared-cm-push")
	cm := inst.(*ChannelManager)

	// Setup Channel
	ch := make(chan types.RuleMsg, 1)
	cm.Register(pipelineID, channelName, ch)
	defer cm.Unregister(pipelineID, channelName)

	// Setup Context and Msg
	ctx := utils.NewMockNodeCtx()
	msg := types.NewMsg("test", "", nil, types.NewDataT())

	// Init
	node.Init(map[string]interface{}{
		CfgPipelineID:    pipelineID,
		CfgChannelName:   channelName,
		CfgBlocking:      true,
		"channelManager": "ref://shared-cm-push",
	})

	// Execute
	node.OnMsg(ctx, msg)

	// Verify
	// If FailureErr is not nil, print it
	if ctx.FailureErr != nil {
		t.Errorf("Unexpected failure: %v", ctx.FailureErr)
	}
	assert.Nil(t, ctx.FailureErr)
	assert.NotNil(t, ctx.SuccessMsg)

	// Verify data in channel
	select {
	case msg := <-ch:
		assert.NotNil(t, msg)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for data in channel")
	}
}

func TestChannelPushNode_ChannelFull_NonBlocking(t *testing.T) {
	node := &ChannelPushNode{}
	pipelineID := "full_pipeline"
	channelName := "full_channel"

	// Setup Shared Channel Manager
	cmDSL := `{"metadata":{"nodes":[{"id":"shared-cm-full","type":"resource/channel_manager","name":"Shared CM Full"}]}}`
	pool := registry.Default.GetSharedNodePool()
	pool.Load([]byte(cmDSL), registry.Default.GetNodeManager())
	inst, _ := pool.GetInstance("shared-cm-full")
	cm := inst.(*ChannelManager)

	// Setup Full Channel
	ch := make(chan types.RuleMsg, 1)
	ch <- types.NewMsg("full", "full", nil, nil)
	cm.Register(pipelineID, channelName, ch)
	defer cm.Unregister(pipelineID, channelName)

	ctx := utils.NewMockNodeCtx()
	node.Init(map[string]interface{}{
		CfgPipelineID:    pipelineID,
		CfgChannelName:   channelName,
		CfgBlocking:      false,
		"channelManager": "ref://shared-cm-full",
	})

	msg := types.NewMsg("test", "", nil, types.NewDataT())

	node.OnMsg(ctx, msg)

	assert.NotNil(t, ctx.FailureErr)
	assert.Contains(t, ctx.FailureErr.Error(), "channel full")
}

func TestChannelPushNode_ChannelNotFound(t *testing.T) {
	node := &ChannelPushNode{}

	// Setup Shared Channel Manager
	cmDSL := `{"metadata":{"nodes":[{"id":"shared-cm-404","type":"resource/channel_manager","name":"Shared CM 404"}]}}`
	pool := registry.Default.GetSharedNodePool()
	pool.Load([]byte(cmDSL), registry.Default.GetNodeManager())

	ctx := utils.NewMockNodeCtx()
	node.Init(map[string]interface{}{
		CfgPipelineID:    "invalid",
		CfgChannelName:   "invalid",
		"channelManager": "ref://shared-cm-404",
	})
	msg := types.NewMsg("test", "", nil, types.NewDataT())

	node.OnMsg(ctx, msg)

	assert.NotNil(t, ctx.FailureErr)
	assert.Contains(t, ctx.FailureErr.Error(), "channel not found")
}

func TestChannelPushNode_DataContract_DynamicRuleMsgConfigReadsAreExplicit(t *testing.T) {
	node := &ChannelPushNode{}
	node.BaseNode = *types.NewBaseNode(ChannelPushNodeType, types.NodeMetadata{})

	err := node.Init(map[string]any{
		CfgPipelineID:  "${rulemsg://dataT/obj_route_pipeline?sid=String}",
		CfgChannelName: "${rulemsg://dataT/obj_route_channel?sid=String}",
	})
	assert.NoError(t, err)

	contract := node.DataContract()
	assert.True(t, containsURI(contract.Reads, "rulemsg://*"))
	assert.True(t, containsURI(contract.Reads, "rulemsg://dataT/obj_route_pipeline?sid=String"))
	assert.True(t, containsURI(contract.Reads, "rulemsg://dataT/obj_route_channel?sid=String"))
	assert.Equal(t, []string{"rulemsg://*"}, contract.Writes)
}

func TestChannelPushNode_DataContract_StaticConfigOnlyKeepsPassThrough(t *testing.T) {
	node := &ChannelPushNode{}
	node.BaseNode = *types.NewBaseNode(ChannelPushNodeType, types.NodeMetadata{})

	err := node.Init(map[string]any{
		CfgPipelineID:  "ep-static-pipeline",
		CfgChannelName: "ch_static_input",
	})
	assert.NoError(t, err)

	contract := node.DataContract()
	assert.Equal(t, []string{"rulemsg://*"}, contract.Reads)
	assert.Equal(t, []string{"rulemsg://*"}, contract.Writes)
}
