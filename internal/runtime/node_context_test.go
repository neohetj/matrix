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

package runtime_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/neohetj/matrix/internal/runtime"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/test/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// setupMockChain 辅助函数：设置mock chain的行为
func setupMockChain(mockChain *utils.MockChainInstance) {
	// 默认返回一个定义的规则链，避免在调用 Logger/ChainID 时发生 panic
	def := &types.RuleChainDef{
		RuleChain: types.RuleChainData{
			ID: "test_chain",
		},
	}
	mockChain.On("Definition").Return(def).Maybe()
}

// createRuntime 创建一个带有 Mock Scheduler 的 DefaultRuntime
// 由于我们在 _test 包中，需要使用公开的构造函数
func createRuntime(t *testing.T, mockScheduler *utils.MockScheduler) *runtime.DefaultRuntime {
	// 我们需要一个虚拟的 chain def 来满足 NewDefaultRuntime 的参数要求，
	// 但我们不会依赖它内部构建的 chain instance。
	// 我们将在调用 NewDefaultNodeCtx 时注入我们自己的 MockChainInstance。
	chainDef := &types.RuleChainDef{
		RuleChain: types.RuleChainData{ID: "dummy"},
		Metadata:  types.MetadataData{},
	}

	r, err := runtime.NewDefaultRuntime(mockScheduler, chainDef)
	assert.NoError(t, err)
	return r
}

// --- Tests ---

// TestTellNext_NoNextNode 测试无后续节点的情况
// 验证：当没有后续连接时，应该立即调用 onEnd 回调（childDone）。
func TestTellNext_NoNextNode(t *testing.T) {
	// Arrange
	mockScheduler := new(utils.MockScheduler)
	mockChain := new(utils.MockChainInstance)
	setupMockChain(mockChain)

	r := createRuntime(t, mockScheduler)

	currentNodeID := "node1"
	selfDef := &types.NodeDef{ID: currentNodeID}

	msg := new(utils.MockRuleMsg)

	// Mock connections: 没有找到连接
	mockChain.On("GetConnections", currentNodeID).Return([]types.Connection{})

	// 追踪 onEnd 回调
	var onEndCalled bool
	onEnd := func(m types.RuleMsg, err error) {
		onEndCalled = true
		assert.Nil(t, err)
	}

	// 注入 mockChain
	ctx := runtime.NewDefaultNodeCtx(context.Background(), r, mockChain, selfDef, nil, onEnd, nil, nil)

	// Act
	ctx.TellNext(msg, "Success")

	// Assert
	assert.True(t, onEndCalled)
	mockChain.AssertExpectations(t)
}

// TestTellNext_NoMatchingRelation 测试关系不匹配的情况
// 验证：虽然有连接，但关系类型不匹配（例如要求 Success 但连接是 Failure），也应结束。
func TestTellNext_NoMatchingRelation(t *testing.T) {
	// Arrange
	mockScheduler := new(utils.MockScheduler)
	mockChain := new(utils.MockChainInstance)
	setupMockChain(mockChain)

	r := createRuntime(t, mockScheduler)

	currentNodeID := "node1"
	selfDef := &types.NodeDef{ID: currentNodeID}

	msg := new(utils.MockRuleMsg)

	// Mock connections: 存在连接，但类型是 "Failure"
	connections := []types.Connection{
		{FromID: "node1", ToID: "node2", Type: "Failure"},
	}
	mockChain.On("GetConnections", currentNodeID).Return(connections)

	var onEndCalled bool
	onEnd := func(m types.RuleMsg, err error) {
		onEndCalled = true
		assert.Nil(t, err)
	}

	ctx := runtime.NewDefaultNodeCtx(context.Background(), r, mockChain, selfDef, nil, onEnd, nil, nil)

	// Act
	// 我们请求 "Success" 关系
	ctx.TellNext(msg, "Success")

	// Assert
	assert.True(t, onEndCalled)
	mockChain.AssertExpectations(t)
}

// TestTellNext_Success 测试成功流转的情况
// 验证：找到下一个节点，提交任务到调度器，并在任务中执行下一个节点的 OnMsg。
func TestTellNext_Success(t *testing.T) {
	// Arrange
	mockScheduler := new(utils.MockScheduler)
	mockChain := new(utils.MockChainInstance)
	setupMockChain(mockChain)

	r := createRuntime(t, mockScheduler)

	currentNodeID := "node1"
	nextNodeID := "node2"
	selfDef := &types.NodeDef{ID: currentNodeID}
	nextDef := &types.NodeDef{ID: nextNodeID, Name: "Node 2"}

	msg := new(utils.MockRuleMsg)
	msg.On("Copy").Return(msg) // 简化起见，Copy 返回自身

	connections := []types.Connection{
		{FromID: currentNodeID, ToID: nextNodeID, Type: "Success"},
	}
	mockChain.On("GetConnections", currentNodeID).Return(connections)

	mockNextNode := new(utils.MockNode)
	mockChain.On("GetNode", nextNodeID).Return(mockNextNode, true)
	mockChain.On("GetNodeDef", nextNodeID).Return(nextDef, true)

	// Scheduler 应该收到任务。我们在测试中立即执行它。
	mockScheduler.On("Submit", mock.Anything).Return(nil, true).Run(func(args mock.Arguments) {
		task := args.Get(0).(func())
		task()
	})

	// 期望下一个节点的 OnMsg 被调用
	mockNextNode.On("OnMsg", mock.Anything, msg).Return()

	ctx := runtime.NewDefaultNodeCtx(context.Background(), r, mockChain, selfDef, nil, nil, nil, nil)

	// Act
	ctx.TellNext(msg, "Success")

	// Assert
	mockChain.AssertExpectations(t)
	mockScheduler.AssertExpectations(t)
	mockNextNode.AssertExpectations(t)
	msg.AssertExpectations(t)
}

// TestTellNext_Aspects 测试切面执行情况
// 验证：在执行节点逻辑前后，切面的 Before 和 After 方法被正确调用。
func TestTellNext_Aspects(t *testing.T) {
	// Arrange
	mockScheduler := new(utils.MockScheduler)
	mockChain := new(utils.MockChainInstance)
	setupMockChain(mockChain)

	r := createRuntime(t, mockScheduler)

	currentNodeID := "node1"
	nextNodeID := "node2"
	selfDef := &types.NodeDef{ID: currentNodeID}
	nextDef := &types.NodeDef{ID: nextNodeID}

	msg := new(utils.MockRuleMsg)
	msg.On("Copy").Return(msg)

	connections := []types.Connection{
		{FromID: currentNodeID, ToID: nextNodeID, Type: "Success"},
	}
	mockChain.On("GetConnections", currentNodeID).Return(connections)

	mockNextNode := new(utils.MockNode)
	mockChain.On("GetNode", nextNodeID).Return(mockNextNode, true)
	mockChain.On("GetNodeDef", nextNodeID).Return(nextDef, true)

	mockScheduler.On("Submit", mock.Anything).Return(nil, true).Run(func(args mock.Arguments) {
		args.Get(0).(func())()
	})

	mockAspect := new(utils.MockAspect)
	// 期望 Before 被调用
	mockAspect.On("Before", mock.Anything, msg).Return(msg, nil)
	// 期望 After 被调用 (OnMsg 没有报错)
	mockAspect.On("After", mock.Anything, msg, nil).Return()

	mockNextNode.On("OnMsg", mock.Anything, msg).Return()

	ctx := runtime.NewDefaultNodeCtx(context.Background(), r, mockChain, selfDef, nil, nil, []types.Aspect{mockAspect}, nil)

	// Act
	ctx.TellNext(msg, "Success")

	// Assert
	mockAspect.AssertExpectations(t)
	mockNextNode.AssertExpectations(t)
}

// TestTellNext_BeforeAspectError 测试前置切面失败的情况
// 验证：如果 Before 切面返回错误，应该跳过 OnMsg 执行，并触发错误处理（调用 After 并带有错误）。
func TestTellNext_BeforeAspectError(t *testing.T) {
	// Arrange
	mockScheduler := new(utils.MockScheduler)
	mockChain := new(utils.MockChainInstance)
	setupMockChain(mockChain)

	r := createRuntime(t, mockScheduler)

	currentNodeID := "node1"
	nextNodeID := "node2"
	selfDef := &types.NodeDef{ID: currentNodeID}
	nextDef := &types.NodeDef{ID: nextNodeID}

	msg := new(utils.MockRuleMsg)
	msg.On("Copy").Return(msg)

	connections := []types.Connection{
		{FromID: currentNodeID, ToID: nextNodeID, Type: "Success"},
	}
	mockChain.On("GetConnections", currentNodeID).Return(connections)

	mockNextNode := new(utils.MockNode)
	mockChain.On("GetNode", nextNodeID).Return(mockNextNode, true)
	mockChain.On("GetNodeDef", nextNodeID).Return(nextDef, true)

	mockScheduler.On("Submit", mock.Anything).Return(nil, true).Run(func(args mock.Arguments) {
		args.Get(0).(func())()
	})

	mockAspect := new(utils.MockAspect)
	expectedErr := errors.New("aspect failure")

	// 期望 Before 失败
	mockAspect.On("Before", mock.Anything, msg).Return(msg, expectedErr)
	// 期望 After 被调用且带有错误
	mockAspect.On("After", mock.Anything, msg, expectedErr).Return()

	// OnMsg 不应被调用

	ctx := runtime.NewDefaultNodeCtx(context.Background(), r, mockChain, selfDef, nil, nil, []types.Aspect{mockAspect}, nil)

	// Act
	ctx.TellNext(msg, "Success")

	// Assert
	mockAspect.AssertExpectations(t)
	mockNextNode.AssertNotCalled(t, "OnMsg", mock.Anything, mock.Anything)
}

// TestTellNext_MultipleNextNodes 测试多个后续节点/分支的情况
// 验证：如果有多个匹配的连接，所有后续节点都应被提交执行。
func TestTellNext_MultipleNextNodes(t *testing.T) {
	// Arrange
	mockScheduler := new(utils.MockScheduler)
	mockChain := new(utils.MockChainInstance)
	setupMockChain(mockChain)

	r := createRuntime(t, mockScheduler)

	currentNodeID := "node1"
	selfDef := &types.NodeDef{ID: currentNodeID}

	msg := new(utils.MockRuleMsg)
	msg.On("Copy").Return(msg)

	connections := []types.Connection{
		{FromID: currentNodeID, ToID: "node2", Type: "Success"},
		{FromID: currentNodeID, ToID: "node3", Type: "Success"},
	}
	mockChain.On("GetConnections", currentNodeID).Return(connections)

	// Setup Node 2
	mockNode2 := new(utils.MockNode)
	mockChain.On("GetNode", "node2").Return(mockNode2, true)
	mockChain.On("GetNodeDef", "node2").Return(&types.NodeDef{ID: "node2"}, true)
	mockNode2.On("OnMsg", mock.Anything, msg).Return()

	// Setup Node 3
	mockNode3 := new(utils.MockNode)
	mockChain.On("GetNode", "node3").Return(mockNode3, true)
	mockChain.On("GetNodeDef", "node3").Return(&types.NodeDef{ID: "node3"}, true)
	mockNode3.On("OnMsg", mock.Anything, msg).Return()

	mockScheduler.On("Submit", mock.Anything).Return(nil, true).Run(func(args mock.Arguments) {
		args.Get(0).(func())()
	}).Times(2) // 期望提交 2 次任务

	ctx := runtime.NewDefaultNodeCtx(context.Background(), r, mockChain, selfDef, nil, nil, nil, nil)

	// Act
	ctx.TellNext(msg, "Success")

	// Assert
	mockChain.AssertExpectations(t)
	mockScheduler.AssertExpectations(t)
	mockNode2.AssertExpectations(t)
	mockNode3.AssertExpectations(t)
}

// TestTellNext_PanicRecovery 测试节点执行Panic恢复的情况
// 验证：如果节点 OnMsg 发生 panic，运行时应捕获并记录错误，不导致整个程序崩溃。
func TestTellNext_PanicRecovery(t *testing.T) {
	// Arrange
	mockScheduler := new(utils.MockScheduler)
	mockChain := new(utils.MockChainInstance)
	setupMockChain(mockChain)

	r := createRuntime(t, mockScheduler)

	currentNodeID := "node1"
	nextNodeID := "node2"
	selfDef := &types.NodeDef{ID: currentNodeID}
	nextDef := &types.NodeDef{ID: nextNodeID, Name: "Node 2"}

	msg := new(utils.MockRuleMsg)
	msg.On("Copy").Return(msg)

	connections := []types.Connection{
		{FromID: currentNodeID, ToID: nextNodeID, Type: "Success"},
	}
	mockChain.On("GetConnections", currentNodeID).Return(connections)

	mockNextNode := new(utils.MockNode)
	mockChain.On("GetNode", nextNodeID).Return(mockNextNode, true)
	mockChain.On("GetNodeDef", nextNodeID).Return(nextDef, true)

	mockScheduler.On("Submit", mock.Anything).Return(nil, true).Run(func(args mock.Arguments) {
		args.Get(0).(func())()
	})

	// 模拟 OnMsg 发生 panic
	mockNextNode.On("OnMsg", mock.Anything, msg).Run(func(args mock.Arguments) {
		panic("simulated panic")
	})

	// 使用 Aspect 验证 panic 后的处理（After 应被调用且带有错误）
	mockAspect := new(utils.MockAspect)
	mockAspect.On("Before", mock.Anything, msg).Return(msg, nil)
	mockAspect.On("After", mock.Anything, msg, mock.MatchedBy(func(err error) bool {
		return err != nil && fmt.Sprintf("%v", err) == "node execution panic: simulated panic"
	})).Return()

	ctx := runtime.NewDefaultNodeCtx(context.Background(), r, mockChain, selfDef, nil, nil, []types.Aspect{mockAspect}, nil)

	// Act
	assert.NotPanics(t, func() {
		ctx.TellNext(msg, "Success")
	})

	// Assert
	mockAspect.AssertExpectations(t)
}

// TestTellNext_DiamondTopology 测试菱形拓扑结构
// 场景: Node1 -> [Node2, Node3] -> Node4
// 验证：Node4 会被执行两次（一次来自 Node2，一次来自 Node3）。
// Matrix 默认行为是每条路径独立执行。如果要合并执行，需要专门的聚合逻辑（TellNext 本身只负责传递）。
func TestTellNext_DiamondTopology(t *testing.T) {
	// Arrange
	mockScheduler := new(utils.MockScheduler)
	mockChain := new(utils.MockChainInstance)
	setupMockChain(mockChain)

	r := createRuntime(t, mockScheduler)

	node1 := "node1"
	node2 := "node2"
	node3 := "node3"
	node4 := "node4"

	// Defs
	def1 := &types.NodeDef{ID: node1}
	def2 := &types.NodeDef{ID: node2}
	def3 := &types.NodeDef{ID: node3}
	def4 := &types.NodeDef{ID: node4}

	msg := new(utils.MockRuleMsg)
	msg.On("Copy").Return(msg)

	// Connections
	// 1 -> 2, 1 -> 3
	conn1 := []types.Connection{
		{FromID: node1, ToID: node2, Type: "Success"},
		{FromID: node1, ToID: node3, Type: "Success"},
	}
	// 2 -> 4
	conn2 := []types.Connection{
		{FromID: node2, ToID: node4, Type: "Success"},
	}
	// 3 -> 4
	conn3 := []types.Connection{
		{FromID: node3, ToID: node4, Type: "Success"},
	}

	mockChain.On("GetConnections", node1).Return(conn1)
	mockChain.On("GetConnections", node2).Return(conn2)
	mockChain.On("GetConnections", node3).Return(conn3)

	// Node lookups
	mockNode2 := new(utils.MockNode)
	mockNode3 := new(utils.MockNode)
	mockNode4 := new(utils.MockNode)

	mockChain.On("GetNode", node2).Return(mockNode2, true)
	mockChain.On("GetNodeDef", node2).Return(def2, true)

	mockChain.On("GetNode", node3).Return(mockNode3, true)
	mockChain.On("GetNodeDef", node3).Return(def3, true)

	mockChain.On("GetNode", node4).Return(mockNode4, true)
	mockChain.On("GetNodeDef", node4).Return(def4, true)

	// Scheduler: 我们手动控制任务执行
	mockScheduler.On("Submit", mock.Anything).Return(nil, true).Run(func(args mock.Arguments) {
		args.Get(0).(func())()
	})

	// 执行流期望:
	// 1. ctx1.TellNext -> 触发 Node2.OnMsg 和 Node3.OnMsg
	// 2. Node2.OnMsg (模拟) 调用 ctx2.TellNext -> 触发 Node4.OnMsg (实例 A)
	// 3. Node3.OnMsg (模拟) 调用 ctx3.TellNext -> 触发 Node4.OnMsg (实例 B)

	// 我们需要模拟 Node2 和 Node3 的 OnMsg 实现来调用 TellNext
	mockNode2.On("OnMsg", mock.Anything, msg).Run(func(args mock.Arguments) {
		ctx := args.Get(0).(types.NodeCtx)
		ctx.TellNext(msg, "Success")
	})
	mockNode3.On("OnMsg", mock.Anything, msg).Run(func(args mock.Arguments) {
		ctx := args.Get(0).(types.NodeCtx)
		ctx.TellNext(msg, "Success")
	})

	// 期望 Node4 被调用两次
	mockNode4.On("OnMsg", mock.Anything, msg).Return().Times(2)

	ctx1 := runtime.NewDefaultNodeCtx(context.Background(), r, mockChain, def1, nil, nil, nil, nil)

	// Act
	ctx1.TellNext(msg, "Success")

	// Assert
	mockChain.AssertExpectations(t)
	mockScheduler.AssertExpectations(t)
	mockNode2.AssertExpectations(t)
	mockNode3.AssertExpectations(t)
	mockNode4.AssertExpectations(t)
}

// TestTellNext_StopPropagation 测试停止传播机制
// 验证：当消息类型为 MsgTypeStopPropagation 时，TellNext 即使存在后续连接，也不应该触发后续节点。
func TestTellNext_StopPropagation(t *testing.T) {
	// Arrange
	mockScheduler := new(utils.MockScheduler)
	mockChain := new(utils.MockChainInstance)
	setupMockChain(mockChain)

	r := createRuntime(t, mockScheduler)

	currentNodeID := "node1"
	nextNodeID := "node2"
	selfDef := &types.NodeDef{ID: currentNodeID}

	// 构造一个带有 StopPropagation 类型的消息
	msg := new(utils.MockRuleMsg)
	msg.On("Type").Return(cnst.MsgTypeStopPropagation)

	// 虽然存在连接
	connections := []types.Connection{
		{FromID: currentNodeID, ToID: nextNodeID, Type: "Success"},
	}
	// GetConnections 可能不会被调用，如果 TellNext 提前返回
	// 使用 Maybe() 允许它被调用或不被调用，或者我们断言它不被调用
	mockChain.On("GetConnections", currentNodeID).Return(connections).Maybe()

	// 追踪 onEnd 回调
	var onEndCalled bool
	onEnd := func(m types.RuleMsg, err error) {
		onEndCalled = true
		assert.Nil(t, err)
	}

	ctx := runtime.NewDefaultNodeCtx(context.Background(), r, mockChain, selfDef, nil, onEnd, nil, nil)

	// Act
	ctx.TellNext(msg, "Success")

	// Assert
	assert.True(t, onEndCalled)
	// Scheduler 不应被调用，因为没有提交新任务
	mockScheduler.AssertNotCalled(t, "Submit", mock.Anything)
}
