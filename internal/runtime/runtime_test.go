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

package runtime

import (
	"bytes"
	"context"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/neohetj/matrix/internal/builtin"
	"github.com/neohetj/matrix/internal/parser"
	"github.com/neohetj/matrix/internal/scheduler"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/message"
	"github.com/neohetj/matrix/pkg/types"
)

func TestRuntime_Execute_LogFunc(t *testing.T) {
	// 1. Arrange
	// Redirect log output to a buffer to capture it
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	// Restore log output at the end of the test
	defer log.SetOutput(os.Stderr)

	p := parser.NewJsonParser()
	s, _ := scheduler.NewAntsScheduler(10)
	defer s.Stop()

	dsl := `
	{
		"ruleChain": {
			"id": "test_chain_2_nodes",
			"name": "Test Chained Log"
		},
		"metadata": {
			"nodes": [
				{
					"id": "log_node_1",
					"type": "functions",
					"name": "First Log Node",
					"configuration": {
						"functionName": "log"
					}
				},
				{
					"id": "log_node_2",
					"type": "functions",
					"name": "Second Log Node",
					"configuration": {
						"functionName": "log"
					}
				}
			],
			"connections": [
				{
					"fromId": "log_node_1",
					"toId": "log_node_2",
					"type": "Success"
				}
			]
		}
	}`

	chainDef, err := p.DecodeRuleChain([]byte(dsl))
	if err != nil {
		t.Fatalf("Failed to decode rule chain: %v", err)
	}

	runtime, err := NewDefaultRuntime(s, chainDef)
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}

	msg := message.NewMsg("TEST_MSG", `{"key":"value"}`, types.Metadata{"source": "test"}, nil).WithDataFormat(cnst.JSON)

	var wg sync.WaitGroup
	wg.Add(1)

	// 2. Act
	// Execute with an empty fromNodeID to let the runtime find the root node.
	err = runtime.Execute(context.Background(), "", msg, func(outMsg types.RuleMsg, err error) {
		defer wg.Done()
		if err != nil {
			t.Errorf("Execution failed with error: %v", err)
		}
	})

	if err != nil {
		t.Fatalf("Failed to start execution: %v", err)
	}

	// Wait for the async execution to complete
	wg.Wait()

	// 3. Assert
	// Give a little time for the log to be written to the buffer
	time.Sleep(10 * time.Millisecond)

	logOutput := logBuf.String()
	// Check that the log function was called twice
	if strings.Count(logOutput, "LOG_FUNC") != 2 {
		t.Errorf("Expected log function to be called 2 times, but was called %d times. Log: %s", strings.Count(logOutput, "LOG_FUNC"), logOutput)
	}
}

func TestRuntime_ExecuteAndWait_LogFunc(t *testing.T) {
	// 1. Arrange
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer log.SetOutput(os.Stderr)

	p := parser.NewJsonParser()
	s, _ := scheduler.NewAntsScheduler(10)
	defer s.Stop()

	dsl := `
	{
		"ruleChain": {
			"id": "test_chain_sync"
		},
		"metadata": {
			"nodes": [
				{
					"id": "log_node_sync_1",
					"type": "functions",
					"configuration": { "functionName": "log" }
				}
			],
			"connections": []
		}
	}`

	chainDef, err := p.DecodeRuleChain([]byte(dsl))
	if err != nil {
		t.Fatalf("Failed to decode rule chain: %v", err)
	}

	runtime, err := NewDefaultRuntime(s, chainDef)
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}

	msg := message.NewMsg("TEST_MSG_SYNC", `{"key":"sync"}`, types.Metadata{"source": "test_sync"}, nil).WithDataFormat(cnst.JSON)

	// 2. Act
	// Execute and wait for the result. Start from the specific node.
	finalMsg, err := runtime.ExecuteAndWait(context.Background(), "log_node_sync_1", msg, nil)

	// 3. Assert
	if err != nil {
		t.Fatalf("ExecuteAndWait failed with error: %v", err)
	}
	if finalMsg == nil {
		t.Fatal("ExecuteAndWait returned a nil message")
	}

	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "LOG_FUNC") {
		t.Errorf("Expected log function to be called, but it wasn't. Log: %s", logOutput)
	}
	if !strings.Contains(logOutput, "sync") {
		t.Errorf("Expected log message to contain 'sync', but it didn't. Log: %s", logOutput)
	}
}

func TestRuntime_Reload(t *testing.T) {
	// 1. Arrange
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer log.SetOutput(os.Stderr)

	p := parser.NewJsonParser()
	s, _ := scheduler.NewAntsScheduler(10)
	defer s.Stop()

	// Version 1 of the DSL
	dslV1 := `
	{
		"metadata": {
			"nodes": [{"id": "log_v1", "type": "functions", "configuration": {"functionName": "log", "business": {"script": "'version 1'"}}}]
		}
	}`
	chainDefV1, _ := p.DecodeRuleChain([]byte(dslV1))

	runtime, err := NewDefaultRuntime(s, chainDefV1)
	if err != nil {
		t.Fatalf("Failed to create runtime v1: %v", err)
	}

	msg := message.NewMsg("TEST_RELOAD", "{}", nil, nil).WithDataFormat(cnst.JSON)

	// 2. Act & Assert for V1
	_, err = runtime.ExecuteAndWait(context.Background(), "log_v1", msg, nil)
	if err != nil {
		t.Fatalf("ExecuteAndWait v1 failed: %v", err)
	}
	if !strings.Contains(logBuf.String(), "version 1") {
		t.Fatalf("Expected log for v1, but not found. Log: %s", logBuf.String())
	}

	// 3. Reload to V2
	logBuf.Reset() // Clear the buffer for the next check

	dslV2 := `
	{
		"metadata": {
			"nodes": [{"id": "log_v2", "type": "functions", "configuration": {"functionName": "log", "business": {"script": "'version 2'"}}}]
		}
	}`
	chainDefV2, _ := p.DecodeRuleChain([]byte(dslV2))

	err = runtime.Reload(chainDefV2)
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	// 4. Act & Assert for V2
	_, err = runtime.ExecuteAndWait(context.Background(), "log_v2", msg, nil)
	if err != nil {
		t.Fatalf("ExecuteAndWait v2 failed: %v", err)
	}
	logOutputV2 := logBuf.String()
	if strings.Contains(logOutputV2, "version 1") {
		t.Fatalf("Found log for v1 after reload. Log: %s", logOutputV2)
	}
	if !strings.Contains(logOutputV2, "version 2") {
		t.Fatalf("Expected log for v2, but not found. Log: %s", logOutputV2)
	}

	// 5. Destroy and Assert
	runtime.Destroy()
	_, err = runtime.ExecuteAndWait(context.Background(), "log_v2", msg, nil)
	if err == nil {
		t.Fatal("Expected error after Destroy, but got nil")
	}
	if !strings.Contains(err.Error(), "destroyed") {
		t.Fatalf("Expected destroyed error, but got: %v", err)
	}
}

// testAspect is a mock Aspect for testing.
type testAspect struct {
	beforeCount int
	afterCount  int
}

func (a *testAspect) Before(ctx types.NodeCtx, msg types.RuleMsg) (types.RuleMsg, error) {
	a.beforeCount++
	newMeta := msg.Metadata()
	newMeta["aspect_before"] = "true"
	msg.SetMetadata(newMeta)
	return msg, nil
}

func (a *testAspect) After(ctx types.NodeCtx, msg types.RuleMsg, err error) {
	a.afterCount++
}

// testCallback is a mock CallbackFunc for testing.
type testCallback struct {
	nodeCompletedCount int
	chainCompleted     bool
	// Store a list of messages for each completed node to inspect its state.
	completedNodes map[string][]types.RuleMsg
	mu             sync.Mutex
}

func (c *testCallback) OnNodeCompleted(ctx types.NodeCtx, msg types.RuleMsg, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.completedNodes == nil {
		c.completedNodes = make(map[string][]types.RuleMsg)
	}
	nodeId := "unknown"
	if selfDef := ctx.SelfDef(); selfDef != nil {
		nodeId = selfDef.ID
	}
	c.completedNodes[nodeId] = append(c.completedNodes[nodeId], msg.Copy())
	c.nodeCompletedCount++
}

func (c *testCallback) OnChainCompleted(msg types.RuleMsg, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.chainCompleted = true
}

func TestRuntime_AOP(t *testing.T) {
	// 1. Arrange
	p := parser.NewJsonParser()
	s, _ := scheduler.NewAntsScheduler(10)
	defer s.Stop()

	dsl := `
	{
		"metadata": {
			"nodes": [
				{"id": "node_a", "type": "functions", "configuration": {"functionName": "log"}},
				{"id": "node_b", "type": "functions", "configuration": {"functionName": "log"}}
			],
			"connections": [{"fromId": "node_a", "toId": "node_b", "type": "Success"}]
		}
	}`
	chainDef, _ := p.DecodeRuleChain([]byte(dsl))

	aspect := &testAspect{}
	callback := &testCallback{}

	runtime, err := NewDefaultRuntime(s, chainDef, WithAspects(aspect), WithCallback(callback))
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}

	msg := message.NewMsg("TEST_AOP", "{}", types.Metadata{}, nil).WithDataFormat(cnst.JSON)

	// 2. Act
	finalMsg, err := runtime.ExecuteAndWait(context.Background(), "node_a", msg, nil)
	if err != nil {
		t.Fatalf("ExecuteAndWait failed: %v", err)
	}

	// 3. Assert
	if finalMsg.Metadata()["aspect_before"] != "true" {
		t.Error("Aspect Before hook did not modify metadata as expected")
	}

	if aspect.beforeCount != 2 {
		t.Errorf("Expected Before to be called 2 times, but got %d", aspect.beforeCount)
	}
	if aspect.afterCount != 2 {
		t.Errorf("Expected After to be called 2 times, but got %d", aspect.afterCount)
	}

	callback.mu.Lock()
	if callback.nodeCompletedCount != 2 {
		t.Errorf("Expected OnNodeCompleted to be called 2 times, but got %d", callback.nodeCompletedCount)
	}
	if !callback.chainCompleted {
		t.Error("Expected OnChainCompleted to be called, but it wasn't")
	}
	callback.mu.Unlock()
}

func TestRuntime_ForkJoin(t *testing.T) {
	// 1. Arrange
	p := parser.NewJsonParser()
	s, _ := scheduler.NewAntsScheduler(10)
	defer s.Stop()

	// This DSL defines a fork/join pattern: entry -> [fork_a, fork_b] -> join
	// fork_a and fork_b will run in parallel. Their scripts will return a JSON string
	// to modify metadata and set the log message.
	dsl := `
	{
		"metadata": {
			"nodes": [
				{"id": "entry", "type": "functions", "configuration": {"functionName": "log"}},
				{"id": "fork_a", "type": "functions", "configuration": {"functionName": "log", "business": {"script": "'{\"log\":\"from a\", \"metadata\":{\"from\":\"a\"}}'"}}},
				{"id": "fork_b", "type": "functions", "configuration": {"functionName": "log", "business": {"script": "'{\"log\":\"from b\", \"metadata\":{\"from\":\"b\"}}'"}}},
				{"id": "join", "type": "functions", "configuration": {"functionName": "log"}}
			],
			"connections": [
				{"fromId": "entry", "toId": "fork_a", "type": "Success"},
				{"fromId": "entry", "toId": "fork_b", "type": "Success"},
				{"fromId": "fork_a", "toId": "join", "type": "Success"},
				{"fromId": "fork_b", "toId": "join", "type": "Success"}
			]
		}
	}`
	chainDef, _ := p.DecodeRuleChain([]byte(dsl))

	// We use a callback to inspect the state of the message at each node's completion.
	callback := &testCallback{}
	runtime, err := NewDefaultRuntime(s, chainDef, WithCallback(callback))
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}

	msg := message.NewMsg("TEST_FORK_JOIN", "{}", types.Metadata{}, nil).WithDataFormat(cnst.JSON)

	// 2. Act
	_, err = runtime.ExecuteAndWait(context.Background(), "entry", msg, nil)
	if err != nil {
		t.Fatalf("ExecuteAndWait failed: %v", err)
	}

	// 3. Assert
	callback.mu.Lock()
	defer callback.mu.Unlock()

	// Assert that the entire chain completed successfully.
	// This proves our waitingCount mechanism works for fork/join.
	if !callback.chainCompleted {
		t.Fatal("Expected OnChainCompleted to be called, but it wasn't")
	}

	// Assert that metadata was isolated for fork_a
	if msgs, ok := callback.completedNodes["fork_a"]; !ok || len(msgs) != 1 {
		t.Fatalf("Expected fork_a to complete exactly once, but got %d completions", len(msgs))
	} else {
		if val := msgs[0].Metadata()["from"]; val != "a" {
			t.Errorf("Expected metadata 'from' to be 'a' for fork_a, but got '%s'", val)
		}
	}

	// Assert that metadata was isolated for fork_b
	if msgs, ok := callback.completedNodes["fork_b"]; !ok || len(msgs) != 1 {
		t.Fatalf("Expected fork_b to complete exactly once, but got %d completions", len(msgs))
	} else {
		if val := msgs[0].Metadata()["from"]; val != "b" {
			t.Errorf("Expected metadata 'from' to be 'b' for fork_b, but got '%s'", val)
		}
	}

	// Assert that join node was completed twice
	if len(callback.completedNodes["join"]) != 2 {
		t.Errorf("Expected join node to be completed twice, but got %d", len(callback.completedNodes["join"]))
	}
}
