package pipeline

import (
	"testing"

	"github.com/neohetj/matrix/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func basePipelineConfig() types.ConfigMap {
	return types.ConfigMap{
		"stages": []any{
			map[string]any{
				"name":      "Stage1",
				"id":        "s1",
				"processor": map[string]any{"id": "chain1", "type": "chain"},
			},
		},
		"exposedChannels": map[string]any{"input": "s1_in"},
		"channelManager":  "ref://shared-cm",
	}
}

func newPipelineEndpointForTest() *PipelineEndpointNode {
	node := &PipelineEndpointNode{}
	node.BaseNode = *types.NewBaseNode(PipelineEndpointNodeType, types.NodeMetadata{})
	return node
}

func TestPipelineEndpointWithoutEnabledStaysAlwaysOn(t *testing.T) {
	node := newPipelineEndpointForTest()
	require.NoError(t, node.Init(basePipelineConfig()))
	assert.Empty(t, node.EnableExpression(), "a pipeline that omits enabled must report no gate")
}

func TestPipelineEndpointKeepsEnabledExpressionVerbatim(t *testing.T) {
	expression := "${config:///pipeline.ingest_enabled?scope=engine,env&default=false}"
	config := basePipelineConfig()
	config["enabled"] = expression

	node := newPipelineEndpointForTest()
	require.NoError(t, node.Init(config))
	assert.Equal(t, expression, node.EnableExpression())
}

func TestPipelineEndpointRejectsMalformedEnabled(t *testing.T) {
	config := basePipelineConfig()
	config["enabled"] = "yes"

	node := newPipelineEndpointForTest()
	err := node.Init(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "enabled")
}
