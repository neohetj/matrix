package pipeline

import (
	"testing"

	"github.com/neohetj/matrix/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestChannelManager(t *testing.T) {
	manager := &ChannelManager{
		channels: make(map[string]chan types.RuleMsg),
	}

	pipelineID := "test_pipeline"
	channelName := "test_channel"
	ch := make(chan types.RuleMsg, 1)

	// Test Register
	manager.Register(pipelineID, channelName, ch)

	// Test Get
	retrievedCh, err := manager.Get(pipelineID, channelName)
	assert.NoError(t, err)
	assert.Equal(t, ch, retrievedCh)

	// Test Get Non-existent
	_, err = manager.Get("invalid", "invalid")
	assert.Error(t, err)

	// Test Unregister
	manager.Unregister(pipelineID, channelName)
	_, err = manager.Get(pipelineID, channelName)
	assert.Error(t, err)
}
