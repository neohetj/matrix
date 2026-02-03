/*
 * Copyright 2025 The Matrix Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package action

import (
	"fmt"
	"testing"

	tutils "github.com/neohetj/matrix/test/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestFlowNode_Init(t *testing.T) {
	t.Run("Valid Configuration", func(t *testing.T) {
		node := flowNodePrototype.New().(*FlowNode)
		config := map[string]interface{}{
			"chainId": "chain-123",
		}
		err := node.Init(config)
		assert.NoError(t, err)
		assert.Equal(t, "chain-123", node.nodeConfig.ChainId)
	})

	t.Run("Missing ChainId", func(t *testing.T) {
		node := flowNodePrototype.New().(*FlowNode)
		// Set ID for better error message in Init
		node.SetID("test-node-id")

		config := map[string]interface{}{}
		err := node.Init(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "'chainId' is not specified")
	})

	t.Run("Invalid Configuration Format", func(t *testing.T) {
		node := flowNodePrototype.New().(*FlowNode)
		config := map[string]interface{}{
			"chainId":      "chain-123",
			"unknownField": "value", // Unknown field should cause error due to ErrorUnused: true
		}
		err := node.Init(config)
		assert.Error(t, err)
	})
}

func TestFlowNode_SubChainTrigger(t *testing.T) {
	node := flowNodePrototype.New().(*FlowNode)
	config := map[string]interface{}{
		"chainId": "chain-123",
	}
	err := node.Init(config)
	assert.NoError(t, err)

	assert.Equal(t, "chain-123", node.GetTargetChainID())
	assert.Empty(t, node.GetInputMapping().MapAll)
	assert.Empty(t, node.GetOutputMapping().MapAll)
}

func TestFlowNode_OnMsg(t *testing.T) {
	// Setup Mocks
	mockRuntimePool := new(tutils.MockRuntimePool)
	mockEngine := new(tutils.MockEngine)
	mockCurrentRuntime := new(tutils.MockRuntime)
	mockTargetRuntime := new(tutils.MockRuntime)

	// Configure mock engine to return mock runtime pool
	mockEngine.On("RuntimePool").Return(mockRuntimePool)
	// Configure current runtime to return mock engine
	mockCurrentRuntime.On("GetEngine").Return(mockEngine)

	node := flowNodePrototype.New().(*FlowNode)
	config := map[string]interface{}{
		"chainId":    "target-chain",
		"fromNodeId": "start-node",
	}
	node.Init(config)

	t.Run("Success Execution", func(t *testing.T) {
		ctx := tutils.NewMockNodeCtx()
		ctx.SetRuntime(mockCurrentRuntime)
		msg := tutils.NewTestRuleMsg()

		mockRuntimePool.On("Get", "target-chain").Return(mockTargetRuntime, true).Once()
		mockTargetRuntime.On("ExecuteAndWait", mock.Anything, "start-node", msg, mock.Anything).Return(msg, nil).Once()

		node.OnMsg(ctx, msg)

		assert.Nil(t, ctx.FailureErr)
		assert.NotNil(t, ctx.SuccessMsg)
	})

	t.Run("Target Chain Not Found", func(t *testing.T) {
		ctx := tutils.NewMockNodeCtx()
		ctx.SetRuntime(mockCurrentRuntime)
		msg := tutils.NewTestRuleMsg()

		mockRuntimePool.On("Get", "target-chain").Return(nil, false).Once()

		node.OnMsg(ctx, msg)

		assert.NotNil(t, ctx.FailureErr)
		assert.Contains(t, ctx.FailureErr.Error(), "target chain with id 'target-chain' not found")
	})

	t.Run("Sub-chain Execution Failed", func(t *testing.T) {
		ctx := tutils.NewMockNodeCtx()
		ctx.SetRuntime(mockCurrentRuntime)
		msg := tutils.NewTestRuleMsg()
		expectedErr := fmt.Errorf("sub-chain error")

		mockRuntimePool.On("Get", "target-chain").Return(mockTargetRuntime, true).Once()
		mockTargetRuntime.On("ExecuteAndWait", mock.Anything, "start-node", msg, mock.Anything).Return(msg, expectedErr).Once()

		node.OnMsg(ctx, msg)

		assert.NotNil(t, ctx.FailureErr)
		assert.Equal(t, expectedErr, ctx.FailureErr)
	})
}
