package pipeline

import (
	"context"
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
	assert.True(t, containsURI(contract.Reads, "rulemsg://dataT/obj_route_pipeline?sid=String"))
	assert.True(t, containsURI(contract.Reads, "rulemsg://dataT/obj_route_channel?sid=String"))
	assert.Empty(t, contract.Writes)
}

func TestChannelPushNode_DataContract_StaticConfigOnlyKeepsLocalDependencies(t *testing.T) {
	node := &ChannelPushNode{}
	node.BaseNode = *types.NewBaseNode(ChannelPushNodeType, types.NodeMetadata{})

	err := node.Init(map[string]any{
		CfgPipelineID:  "ep-static-pipeline",
		CfgChannelName: "ch_static_input",
	})
	assert.NoError(t, err)

	contract := node.DataContract()
	assert.Empty(t, contract.Reads)
	assert.Empty(t, contract.Writes)
}

func TestChannelPushNode_ProjectsMessageToDownstreamRequiredInputs(t *testing.T) {
	node := &ChannelPushNode{}
	node.BaseNode = *types.NewBaseNode(ChannelPushNodeType, types.NodeMetadata{})

	pipelineID := "ep-image-save"
	channelName := "ch_image_save_input"
	targetRuleChainID := "sellitx/rc-image-save"

	sharedDSL := `{
		"metadata": {
			"nodes": [
				{"id":"shared-cm-projection","type":"resource/channel_manager","name":"Shared CM Projection"},
				{
					"id":"ep-image-save",
					"type":"endpoint/pipeline",
					"name":"Image Save Pipeline",
					"configuration":{
						"stages":[
							{
								"name":"Image Save Stage",
								"id":"stage-image-save",
								"concurrency":1,
								"processor":{"id":"sellitx/rc-image-save","type":"chain"},
								"inputChannel":"ch_image_save_input"
							}
						],
						"exposedChannels":{"ch_image_save_input":"ch_image_save_input"},
						"channelManager":"ref://shared-cm-projection"
					}
				}
			]
		}
	}`
	pool := registry.Default.GetSharedNodePool()
	_, err := pool.Load([]byte(sharedDSL), registry.Default.GetNodeManager())
	assert.NoError(t, err)

	inst, err := pool.GetInstance("shared-cm-projection")
	assert.NoError(t, err)
	cm := inst.(*ChannelManager)

	ch := make(chan types.RuleMsg, 1)
	cm.Register(pipelineID, channelName, ch)
	defer cm.Unregister(pipelineID, channelName)

	sourceRuntime := &testProjectionRuntime{
		engine: &utils.MockEngine{
			RuntimePoolValue: &utils.MockRuntimePool{
				Runtimes: map[string]types.Runtime{
					targetRuleChainID: &testProjectionRuntime{
						projection: types.RuleChainCoreObjAnalysis{
							RequiredInputs: types.CoreObjSet{ObjIDs: []string{"image_push_items"}},
						},
					},
				},
			},
			SharedNodePoolValue: pool,
		},
	}
	ctx := utils.NewMockNodeCtx()
	ctx.SetRuntime(sourceRuntime)

	dataT := types.NewDataT()
	dataT.Set("image_push_items", &testCoreObj{key: "image_push_items", body: "keep"})
	dataT.Set("ttscrapedposts", &testCoreObj{key: "ttscrapedposts", body: "drop"})
	msg := types.NewMsg("test", "", nil, dataT)

	err = node.Init(map[string]any{
		CfgPipelineID:    pipelineID,
		CfgChannelName:   channelName,
		CfgBlocking:      true,
		"channelManager": "ref://shared-cm-projection",
	})
	assert.NoError(t, err)

	node.OnMsg(ctx, msg)

	assert.Nil(t, ctx.FailureErr)
	assert.NotNil(t, ctx.SuccessMsg)

	select {
	case pushedMsg := <-ch:
		_, hasImagePushItems := pushedMsg.DataT().Get("image_push_items")
		_, hasScrapedPosts := pushedMsg.DataT().Get("ttscrapedposts")
		assert.True(t, hasImagePushItems)
		assert.False(t, hasScrapedPosts)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for projected message in channel")
	}
}

type testProjectionRuntime struct {
	engine        types.MatrixEngine
	projection    types.RuleChainCoreObjAnalysis
	executeFn     func(context.Context, string, types.RuleMsg, func(types.RuleMsg, error)) error
	definition    *types.RuleChainDef
	chainInstance types.ChainInstance
}

func (r *testProjectionRuntime) Execute(ctx context.Context, fromNodeID string, msg types.RuleMsg, onEnd func(types.RuleMsg, error)) error {
	if r.executeFn != nil {
		return r.executeFn(ctx, fromNodeID, msg, onEnd)
	}
	return nil
}

func (r *testProjectionRuntime) ExecuteAndWait(context.Context, string, types.RuleMsg, func(types.RuleMsg, error)) (types.RuleMsg, error) {
	return nil, nil
}

func (r *testProjectionRuntime) Reload(*types.RuleChainDef) error { return nil }
func (r *testProjectionRuntime) Destroy()                         {}
func (r *testProjectionRuntime) Definition() *types.RuleChainDef  { return r.definition }
func (r *testProjectionRuntime) GetNodePool() types.NodePool      { return nil }
func (r *testProjectionRuntime) GetEngine() types.MatrixEngine    { return r.engine }
func (r *testProjectionRuntime) GetChainInstance() types.ChainInstance {
	return r.chainInstance
}
func (r *testProjectionRuntime) CoreObjProjection() types.RuleChainCoreObjAnalysis {
	return r.projection
}
func (r *testProjectionRuntime) LiveObjectsForEdge(string, string) (types.CoreObjSet, bool) {
	return types.CoreObjSet{}, false
}

type testCoreObj struct {
	key  string
	body any
}

func (o *testCoreObj) Key() string                  { return o.key }
func (o *testCoreObj) Definition() types.CoreObjDef { return nil }
func (o *testCoreObj) Body() any                    { return o.body }
func (o *testCoreObj) SetBody(body any) error {
	o.body = body
	return nil
}
func (o *testCoreObj) DeepCopy() (types.CoreObj, error) {
	return &testCoreObj{key: o.key, body: o.body}, nil
}
