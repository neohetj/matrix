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
	"context"
	"fmt"
	"sync"

	"gitlab.com/neohet/matrix/internal/log"

	"gitlab.com/neohet/matrix/pkg/registry"
	"gitlab.com/neohet/matrix/pkg/types"
)

// Option is a function type for configuring a DefaultRuntime.
type Option func(*DefaultRuntime)

// WithAspects adds Aspects to the runtime.
func WithAspects(aspects ...types.Aspect) Option {
	return func(r *DefaultRuntime) {
		r.aspects = append(r.aspects, aspects...)
	}
}

// WithCallback adds a CallbackFunc to the runtime.
func WithCallback(callback types.CallbackFunc) Option {
	return func(r *DefaultRuntime) {
		r.callback = callback
	}
}

// WithNodePool adds a NodePool to the runtime for managing shared nodes.
func WithNodePool(pool types.NodePool) Option {
	return func(r *DefaultRuntime) {
		r.nodePool = pool
	}
}

// WithNodeManager sets a custom NodeManager for the runtime.
// If not used, the runtime defaults to registry.Default.NodeManager.
func WithNodeManager(nm types.NodeManager) Option {
	return func(r *DefaultRuntime) {
		r.nodeManager = nm
	}
}

// WithLogger is an option that sets a specific logger for a Runtime instance.
// This will override the global logger for this specific runtime.
func WithLogger(logger types.Logger) Option {
	return func(r *DefaultRuntime) {
		r.logger = logger
	}
}

// DefaultRuntime is the default implementation of the Runtime interface.
// It orchestrates the execution of a rule chain.
type DefaultRuntime struct {
	mutex         sync.RWMutex
	chainDef      *types.RuleChainDef
	chainInstance *ChainInstance
	scheduler     types.Scheduler
	nodeManager   types.NodeManager
	nodePool      types.NodePool
	aspects       []types.Aspect
	callback      types.CallbackFunc
	logger        types.Logger
}

// NewDefaultRuntime creates a new, stateful instance of DefaultRuntime.
// It builds the initial chain instance from the provided definition and applies any options.
func NewDefaultRuntime(s types.Scheduler, chainDef *types.RuleChainDef, opts ...Option) (*DefaultRuntime, error) {
	r := &DefaultRuntime{
		scheduler:   s,
		chainDef:    chainDef,
		nodeManager: registry.Default.NodeManager,    // Default to the global manager.
		nodePool:    registry.Default.SharedNodePool, // Default to the global pool.
		logger:      log.GetLogger(),                 // Default to the global logger.
	}

	for _, opt := range opts {
		opt(r)
	}
	// If WithLogger was provided, r.logger is now updated.
	// If custom managers/pools were provided via options, they will overwrite the defaults.

	// If a custom node pool was provided via options, it will overwrite the default.

	instance, err := r.buildChainInstance(chainDef)
	if err != nil {
		return nil, fmt.Errorf("failed to build initial chain instance: %w", err)
	}
	r.chainInstance = instance

	return r, nil
}

// Execute runs the rule chain with the given message.
// It is safe for concurrent use.
func (r *DefaultRuntime) Execute(ctx context.Context, fromNodeID string, msg types.RuleMsg, onEnd func(msg types.RuleMsg, err error)) error {
	// Acquire a read lock to ensure the chainInstance is not being reloaded while we use it.
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if r.chainInstance == nil {
		return fmt.Errorf("runtime is not initialized or has been destroyed")
	}

	// Wrap the original onEnd callback to include the OnChainCompleted hook.
	wrappedOnEnd := func(finalMsg types.RuleMsg, finalErr error) {
		if r.callback != nil {
			r.callback.OnChainCompleted(finalMsg, finalErr)
		}
		if onEnd != nil {
			onEnd(finalMsg, finalErr)
		}
	}

	// 1. Determine the starting node(s).
	var rootNodeIDs []string
	if fromNodeID != "" {
		// Start from a specific node if provided.
		if _, ok := r.chainInstance.nodes[fromNodeID]; !ok {
			return fmt.Errorf("fromNodeID '%s' not found in rule chain", fromNodeID)
		}
		rootNodeIDs = []string{fromNodeID}
	} else {
		// Otherwise, find all nodes with no incoming connections.
		rootNodeIDs = findRootNodes(r.chainInstance)
		if len(rootNodeIDs) == 0 && len(r.chainInstance.nodes) > 0 {
			return fmt.Errorf("no root node found in the rule chain")
		}
	}

	// 2. Create a root context to track the completion of all starting nodes.
	// This root context doesn't represent a real node but acts as a parent for all actual root nodes.
	// Its parent is nil, and it holds the final onEnd callback and AOP components.
	rootCtx := NewDefaultNodeCtx(ctx, r, r.chainInstance, &types.NodeDef{ID: "root"},
		nil, wrappedOnEnd, r.aspects, r.callback)

	// If there are no nodes to execute, end the process immediately.
	if len(rootNodeIDs) == 0 {
		wrappedOnEnd(msg, nil)
		return nil
	}

	// 3. Execute all actual root nodes asynchronously.
	for _, rootNodeID := range rootNodeIDs {
		nodeInstance := r.chainInstance.nodes[rootNodeID]
		nodeDef := r.chainInstance.nodeDefs[rootNodeID]

		// Signal that the root context should wait for this new branch to complete.
		rootCtx.childReady()

		// Create a context for this specific node execution, with rootCtx as its parent.
		// TODO: NodeCtx实现是否绑定到Runtime中，使其可以加载使用不同的NodeCtx
		nodeCtx := NewDefaultNodeCtx(ctx, r, r.chainInstance, nodeDef,
			rootCtx, wrappedOnEnd, r.aspects, r.callback)

		// Submit the execution task to the scheduler.
		if err := r.scheduler.Submit(func() {
			// Defer the After aspect calls
			var onMsgErr error
			defer func() {
				for _, aspect := range nodeCtx.aspects {
					aspect.After(nodeCtx, msg, onMsgErr)
				}
			}()

			// 1. Execute Before aspects
			var processedMsg = msg
			for _, aspect := range nodeCtx.aspects {
				var err error
				processedMsg, err = aspect.Before(nodeCtx, processedMsg)
				if err != nil {
					onMsgErr = err
					// If Before aspect fails, skip OnMsg and signal completion immediately
					nodeCtx.childDone(processedMsg, onMsgErr)
					return
				}
			}

			// 2. Execute the node's OnMsg
			nodeInstance.OnMsg(nodeCtx, processedMsg)
		}); err != nil {
			// If task submission fails, we must decrement the waiting counter
			// to prevent the chain from getting stuck waiting for a task that never started.
			rootCtx.childDone(msg, err)
			fmt.Printf("Error submitting task for node %s: %v\n", rootNodeID, err)
		}
	}

	// The Execute function returns quickly. The actual result is handled by the onEnd callback,
	// which should be triggered by the NodeCtx's Tell... methods.
	return nil
}

// buildChainInstance creates a live instance of the rule chain from its definition.
func (r *DefaultRuntime) buildChainInstance(chainDef *types.RuleChainDef) (*ChainInstance, error) {
	instance := &ChainInstance{
		def:         chainDef,
		nodes:       make(map[string]types.Node),
		nodeDefs:    make(map[string]*types.NodeDef),
		connections: make(map[string][]types.Connection),
	}

	// Initialize all nodes defined in the DSL.
	for _, nodeDef := range chainDef.Metadata.Nodes {
		def := nodeDef // Create a new variable for the loop to avoid closure issues.
		var node types.Node
		var err error

		// Check if a shared node instance exists in the pool.
		if r.nodePool != nil {
			if sharedCtx, ok := r.nodePool.Get(def.ID); ok {
				node = sharedCtx.GetNode()
			}
		}

		// If not found in the pool, create a new instance.
		if node == nil {
			node, err = r.nodeManager.NewNode(types.NodeType(def.Type))
			if err != nil {
				return nil, err
			}

			// V2 Initialization Flow:
			// 1. Set instance-specific metadata.
			node.SetID(def.ID)
			node.SetName(def.Name)

			// 2. Initialize with business-specific configuration only.
			// TODO: 增加拓扑，以方便检查是否有参数未设置或者连接不正确
			if err = node.Init(def.Configuration); err != nil {
				return nil, fmt.Errorf("failed to initialize node '%s': %w", def.ID, err)
			}
		}

		instance.nodes[def.ID] = node
		instance.nodeDefs[def.ID] = &def
	}

	// Build the connection map for easy lookup.
	for _, conn := range chainDef.Metadata.Connections {
		instance.connections[conn.FromID] = append(instance.connections[conn.FromID], conn)
	}

	return instance, nil
}

// findRootNodes identifies all nodes with no incoming connections.
func findRootNodes(instance *ChainInstance) []string {
	inDegrees := make(map[string]int)
	for id := range instance.nodes {
		inDegrees[id] = 0
	}

	for _, conns := range instance.connections {
		for _, conn := range conns {
			inDegrees[conn.ToID]++
		}
	}

	var rootNodes []string
	for id, degree := range inDegrees {
		if degree == 0 {
			rootNodes = append(rootNodes, id)
		}
	}
	return rootNodes
}

// ExecuteAndWait runs the rule chain synchronously and waits for its completion.
// It accepts an optional `onEnd` callback which will be executed before the function returns.
func (r *DefaultRuntime) ExecuteAndWait(ctx context.Context, fromNodeID string, msg types.RuleMsg, onEnd func(msg types.RuleMsg, err error)) (types.RuleMsg, error) {
	// doneChan is used to block until the internalOnEnd callback is fired.
	doneChan := make(chan struct {
		msg types.RuleMsg
		err error
	}, 1)

	// internalOnEnd is a new callback that decorates the user-provided onEnd.
	internalOnEnd := func(finalMsg types.RuleMsg, finalErr error) {
		// First, execute the user's callback if it exists.
		if onEnd != nil {
			onEnd(finalMsg, finalErr)
		}
		// Then, notify the channel to unblock ExecuteAndWait.
		doneChan <- struct {
			msg types.RuleMsg
			err error
		}{msg: finalMsg, err: finalErr}
		close(doneChan)
	}
	// TODO: 不要在Execute中传OnEnd？有没有更优雅的方案
	// Start the asynchronous execution with the decorated callback.
	if err := r.Execute(ctx, fromNodeID, msg, internalOnEnd); err != nil {
		// Handle immediate errors from Execute, e.g., invalid chain definition.
		return nil, err
	}

	// Wait for either the execution to complete or the context to be done.
	select {
	case <-ctx.Done():
		// The context was canceled or timed out.
		return nil, ctx.Err()
	case result := <-doneChan:
		// The execution finished, return its result.
		return result.msg, result.err
	}
}

// Reload atomically replaces the underlying rule chain instance with a new one.
func (r *DefaultRuntime) Reload(newChainDef *types.RuleChainDef) error {
	// 1. Build the new instance. This is done outside the lock to avoid
	//    holding the lock during a potentially long operation.
	newInstance, err := r.buildChainInstance(newChainDef)
	if err != nil {
		return fmt.Errorf("failed to build new chain instance for reload: %w", err)
	}

	// 2. Acquire a write lock to atomically swap the instances.
	r.mutex.Lock()
	oldInstance := r.chainInstance
	r.chainInstance = newInstance
	r.chainDef = newChainDef // Update the definition as well
	r.mutex.Unlock()

	// 3. Destroy the old instance after the swap is complete.
	//    This ensures that new executions use the new instance, while
	//    ongoing executions on the old instance can finish gracefully.
	if oldInstance != nil {
		for _, node := range oldInstance.nodes {
			node.Destroy()
		}
	}

	return nil
}

// Destroy releases all resources held by the runtime.
func (r *DefaultRuntime) Destroy() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.chainInstance != nil {
		for _, node := range r.chainInstance.nodes {
			node.Destroy()
		}
		r.chainInstance = nil // Mark as destroyed
	}
}

// Definition returns the underlying rule chain definition.
func (r *DefaultRuntime) Definition() *types.RuleChainDef {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.chainDef
}

// GetNodePool implements the types.Runtime interface.
func (r *DefaultRuntime) GetNodePool() types.NodePool {
	return r.nodePool
}
