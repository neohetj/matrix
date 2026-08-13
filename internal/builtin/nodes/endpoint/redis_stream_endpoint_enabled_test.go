package endpoint

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func baseRedisStreamConfig() map[string]any {
	return map[string]any{
		"redisClient": "ref://redis",
		"stream":      "events",
		"group":       "projector",
		"ruleChainId": "project-event",
	}
}

func TestRedisStreamEndpointWithoutEnabledStaysAlwaysOn(t *testing.T) {
	node := &RedisStreamEndpointNode{}
	require.NoError(t, node.Init(baseRedisStreamConfig()))
	assert.Empty(t, node.EnableExpression(), "an endpoint that omits enabled must report no gate")
}

func TestRedisStreamEndpointKeepsEnabledExpressionVerbatim(t *testing.T) {
	expression := "${config:///blokx.fact_sync.consumers_enabled?scope=engine,env&default=false}"
	cfg := baseRedisStreamConfig()
	cfg["enabled"] = expression

	node := &RedisStreamEndpointNode{}
	require.NoError(t, node.Init(cfg))
	assert.Equal(t, expression, node.EnableExpression(),
		"the node must not resolve the expression itself; the engine owns resolution")
}

func TestRedisStreamEndpointAcceptsBooleanLiterals(t *testing.T) {
	for _, literal := range []string{"true", "false"} {
		cfg := baseRedisStreamConfig()
		cfg["enabled"] = literal

		node := &RedisStreamEndpointNode{}
		require.NoError(t, node.Init(cfg), "literal %q must be accepted", literal)
		assert.Equal(t, literal, node.EnableExpression())
	}
}

func TestRedisStreamEndpointRejectsMalformedEnabled(t *testing.T) {
	for _, value := range []string{"yes", "TRUE", " true", "true ", "${}"} {
		cfg := baseRedisStreamConfig()
		cfg["enabled"] = value

		node := &RedisStreamEndpointNode{}
		err := node.Init(cfg)
		require.Error(t, err, "value %q must be rejected at init time", value)
		assert.Contains(t, err.Error(), "enabled")
	}
}
