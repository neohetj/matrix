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

package action_test

import (
	"context"
	"testing"
	"time"

	"github.com/neohetj/matrix/internal/builtin/nodes/action"
	"github.com/neohetj/matrix/internal/contract"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/test/utils"
	"github.com/stretchr/testify/assert"
)

func TestAggregatorNode_OnMsg_Success(t *testing.T) {
	node := &action.AggregatorNode{}
	node.SetID("agg-node")

	// Setup topology
	chainDef := &types.RuleChainDef{
		Metadata: types.MetadataData{
			Connections: []types.Connection{
				{FromID: "node1", ToID: "agg-node"},
				{FromID: "node2", ToID: "agg-node"},
				{FromID: "node3", ToID: "agg-node"},
			},
		},
	}
	mockChain := new(utils.MockChainInstance)
	mockChain.On("Definition").Return(chainDef)

	mockRuntime := new(utils.MockRuntime)
	mockRuntime.ChainInstance = mockChain

	// Shared DataT
	dataT := contract.NewDataT()

	// 1. First Message from node1
	ctx1 := utils.NewMockNodeCtx()
	ctx1.NodeIDValue = "agg-node"
	ctx1.PreviousNodeIDValue = "node1"
	ctx1.SetRuntime(mockRuntime)

	msg1 := contract.NewDefaultRuleMsg("TEST", "{}", nil, dataT)

	node.OnMsg(ctx1, msg1)
	assert.Nil(t, ctx1.SuccessMsg)

	// 2. Second Message from node2
	ctx2 := utils.NewMockNodeCtx()
	ctx2.NodeIDValue = "agg-node"
	ctx2.PreviousNodeIDValue = "node2"
	ctx2.SetRuntime(mockRuntime)

	msg2 := contract.NewDefaultRuleMsg("TEST", "{}", nil, dataT) // Share DataT!

	node.OnMsg(ctx2, msg2)
	assert.Nil(t, ctx2.SuccessMsg)

	// 3. Duplicate Message from node1 (should be ignored)
	node.OnMsg(ctx1, msg1)
	assert.Nil(t, ctx1.SuccessMsg)

	// 4. Third Message from node3
	ctx3 := utils.NewMockNodeCtx()
	ctx3.NodeIDValue = "agg-node"
	ctx3.PreviousNodeIDValue = "node3"
	ctx3.SetRuntime(mockRuntime)

	msg3 := contract.NewDefaultRuleMsg("TEST", "{}", nil, dataT)

	node.OnMsg(ctx3, msg3)
	assert.NotNil(t, ctx3.SuccessMsg)
	assert.Equal(t, msg3, ctx3.SuccessMsg)
}

func TestAggregatorNode_OnMsg_Timeout(t *testing.T) {
	node := &action.AggregatorNode{}
	node.SetID("agg-node")

	// Setup topology (2 predecessors)
	chainDef := &types.RuleChainDef{
		Metadata: types.MetadataData{
			Connections: []types.Connection{
				{FromID: "node1", ToID: "agg-node"},
				{FromID: "node2", ToID: "agg-node"},
			},
		},
	}
	mockChain := new(utils.MockChainInstance)
	mockChain.On("Definition").Return(chainDef)

	mockRuntime := new(utils.MockRuntime)
	mockRuntime.ChainInstance = mockChain

	// Config
	config := map[string]any{"timeout": "100ms"}

	// Shared DataT
	dataT := contract.NewDataT()

	// 1. First Message from node1
	ctx1 := utils.NewMockNodeCtx(utils.WithTestNodeConfig(config))
	ctx1.NodeIDValue = "agg-node"
	ctx1.PreviousNodeIDValue = "node1"
	ctx1.SetRuntime(mockRuntime)
	ctx1.Ctx = context.Background()

	msg1 := contract.NewDefaultRuleMsg("TEST", "{}", nil, dataT)

	node.OnMsg(ctx1, msg1)

	// Wait for timeout
	time.Sleep(200 * time.Millisecond)

	assert.NotNil(t, ctx1.FailureErr)
	assert.Contains(t, ctx1.FailureErr.Error(), "timed out")
}

func TestAggregatorNode_OnMsg_IgnoreLateMessages(t *testing.T) {
	node := &action.AggregatorNode{}
	node.SetID("agg-node")

	// Setup topology (1 predecessor - immediate success)
	chainDef := &types.RuleChainDef{
		Metadata: types.MetadataData{
			Connections: []types.Connection{
				{FromID: "node1", ToID: "agg-node"},
			},
		},
	}
	mockChain := new(utils.MockChainInstance)
	mockChain.On("Definition").Return(chainDef)

	mockRuntime := new(utils.MockRuntime)
	mockRuntime.ChainInstance = mockChain

	// Shared DataT
	dataT := contract.NewDataT()

	// 1. First Message
	ctx1 := utils.NewMockNodeCtx()
	ctx1.NodeIDValue = "agg-node"
	ctx1.PreviousNodeIDValue = "node1"
	ctx1.SetRuntime(mockRuntime)

	msg1 := contract.NewDefaultRuleMsg("TEST", "{}", nil, dataT)

	node.OnMsg(ctx1, msg1)
	assert.NotNil(t, ctx1.SuccessMsg)

	// 2. Second Message (Duplicate/Late from same sender)
	// Should be ignored (TellSuccess not called again on this ctx, but ctx1 already has SuccessMsg set)
	// Use new context to verify
	ctx2 := utils.NewMockNodeCtx()
	ctx2.NodeIDValue = "agg-node"
	ctx2.PreviousNodeIDValue = "node1"
	ctx2.SetRuntime(mockRuntime)

	node.OnMsg(ctx2, msg1)
	assert.Nil(t, ctx2.SuccessMsg) // Should be "Wait"
}
