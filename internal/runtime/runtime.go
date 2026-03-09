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
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/neohetj/matrix/internal/log"

	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/asset"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/rulechain"
	"github.com/neohetj/matrix/pkg/types"
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

// WithEngine sets the engine instance for the runtime.
func WithEngine(e types.MatrixEngine) Option {
	return func(r *DefaultRuntime) {
		r.engine = e
		r.nodeManager = e.NodeManager()
		r.nodePool = e.SharedNodePool()
		r.logger = e.Logger()
	}
}

// DefaultRuntime is the default implementation of the Runtime interface.
// It orchestrates the execution of a rule chain.
type DefaultRuntime struct {
	mutex         sync.RWMutex
	chainDef      *types.RuleChainDef
	chainInstance types.ChainInstance
	scheduler     types.Scheduler
	nodeManager   types.NodeManager
	nodePool      types.NodePool
	aspects       []types.Aspect
	callback      types.CallbackFunc
	logger        types.Logger
	engine        types.MatrixEngine
	coreObjPlan   types.RuleChainCoreObjAnalysis
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
	r.coreObjPlan = rulechain.AnalyzeCoreObjProjection(chainDef, instance)

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

	onErrorCfg := r.resolveOnErrorConfig()

	// Wrap the original onEnd callback to include the OnChainCompleted hook.
	wrappedOnEnd := func(finalMsg types.RuleMsg, finalErr error) {
		resolvedMsg, resolvedErr := r.handleOnErrorStrategy(ctx, msg, finalMsg, finalErr, onErrorCfg)
		if r.callback != nil {
			r.callback.OnChainCompleted(resolvedMsg, resolvedErr)
		}
		if onEnd != nil {
			onEnd(resolvedMsg, resolvedErr)
		}
	}

	// 1. Determine the starting node(s).
	var rootNodeIDs []string
	if fromNodeID != "" {
		// Start from a specific node if provided.
		if _, ok := r.chainInstance.GetNode(fromNodeID); !ok {
			return fmt.Errorf("fromNodeID '%s' not found in rule chain", fromNodeID)
		}
		rootNodeIDs = []string{fromNodeID}
	} else {
		// Otherwise, find all nodes with no incoming connections.
		rootNodeIDs = r.chainInstance.GetRootNodeIDs()
		if len(rootNodeIDs) == 0 && len(r.chainInstance.GetAllNodes()) > 0 {
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
		nodeInstance, okNode := r.chainInstance.GetNode(rootNodeID)
		nodeDef, okDef := r.chainInstance.GetNodeDef(rootNodeID)

		if !okNode || !okDef {
			// This should theoretically not happen if validation passed, but for safety:
			rootCtx.childDone(msg, fmt.Errorf("root node '%s' not found", rootNodeID))
			continue
		}

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
				if r := recover(); r != nil {
					onMsgErr = fmt.Errorf("node execution panic: %v", r)
					nodeCtx.Error("Recovered from panic in node execution", "panic", r)
					nodeCtx.childDone(msg, onMsgErr)
				}
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

func (r *DefaultRuntime) resolveOnErrorConfig() *types.RuleChainOnError {
	if r.chainDef == nil {
		return nil
	}
	return r.chainDef.RuleChain.OnError
}

func (r *DefaultRuntime) resolveOnErrorStrategy(cfg *types.RuleChainOnError) types.RuleChainOnErrorStrategy {
	if cfg == nil || cfg.Strategy == "" {
		return types.RuleChainOnErrorStrategyHalt
	}
	switch cfg.Strategy {
	case types.RuleChainOnErrorStrategyContinue, types.RuleChainOnErrorStrategyRedirect:
		return cfg.Strategy
	default:
		return types.RuleChainOnErrorStrategyHalt
	}
}

func (r *DefaultRuntime) ensureErrorMetadata(msg types.RuleMsg, err error) types.RuleMsg {
	if msg == nil || err == nil {
		return msg
	}
	md := msg.Metadata()
	if md == nil {
		md = types.Metadata{}
	}
	md[types.MetaError] = err.Error()
	md[types.MetaErrorTimestamp] = time.Now().UTC().Format(time.RFC3339)
	var fault *types.Fault
	if errors.As(err, &fault) {
		md[types.MetaErrorCode] = string(fault.Code)
	}
	msg.SetMetadata(md)
	return msg
}

func (r *DefaultRuntime) handleOnErrorStrategy(ctx context.Context, inMsg types.RuleMsg, finalMsg types.RuleMsg, finalErr error, cfg *types.RuleChainOnError) (types.RuleMsg, error) {
	if finalErr == nil {
		return finalMsg, nil
	}

	effectiveMsg := finalMsg
	if effectiveMsg == nil {
		effectiveMsg = inMsg
	}
	effectiveMsg = r.ensureErrorMetadata(effectiveMsg, finalErr)

	strategy := r.resolveOnErrorStrategy(cfg)
	switch strategy {
	case types.RuleChainOnErrorStrategyContinue:
		return effectiveMsg, nil
	case types.RuleChainOnErrorStrategyRedirect:
		if cfg == nil || cfg.Handler == "" {
			return effectiveMsg, finalErr
		}
		if r.engine == nil || r.engine.RuntimePool() == nil {
			return effectiveMsg, finalErr
		}
		handlerRT, ok := r.engine.RuntimePool().Get(cfg.Handler)
		if !ok {
			return effectiveMsg, finalErr
		}
		redirectMsg, redirectErr := handlerRT.ExecuteAndWait(ctx, "", effectiveMsg, nil)
		if redirectErr != nil {
			return effectiveMsg, finalErr
		}
		return redirectMsg, nil
	case types.RuleChainOnErrorStrategyHalt:
		fallthrough
	default:
		return effectiveMsg, finalErr
	}
}

// buildChainInstance creates a live instance of the rule chain from its definition.
func (r *DefaultRuntime) buildChainInstance(chainDef *types.RuleChainDef) (types.ChainInstance, error) {
	instance := NewDefaultChainInstance(chainDef)

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

		if bindable, ok := node.(types.NodeDefBinding); ok {
			bindable.BindNodeDef(&def)
		}

		instance.AddNode(node, &def)
	}

	// Build the connection map for easy lookup.
	for _, conn := range chainDef.Metadata.Connections {
		instance.AddConnection(conn)
	}

	// Validate data contracts between connected nodes.
	if err := r.validateDataContract(instance); err != nil {
		return nil, fmt.Errorf("data contract validation failed: %w", err)
	}

	return instance, nil
}

func (r *DefaultRuntime) validateDataContract(instance types.ChainInstance) error {
	nodes := instance.GetAllNodes()
	// Map nodeID -> writes
	nodeWrites := make(map[string]map[string]cnst.MFormat)

	parseDataURI := func(uri string) (string, cnst.MFormat, bool) {
		parsed, err := asset.ParseRuleMsg(uri)
		if err != nil {
			return "", cnst.UNKNOWN, false
		}
		if parsed.Scheme != cnst.DATA {
			return "", cnst.UNKNOWN, false
		}
		formatVal := parsed.Query.Get("format")
		if formatVal == "" {
			return parsed.Path, cnst.UNKNOWN, false
		}
		format := cnst.MFormat(formatVal)
		if !format.IsValid() {
			return parsed.Path, cnst.UNKNOWN, true
		}
		return parsed.Path, format, true
	}

	// 1. Collect writes
	for id, node := range nodes {
		contract := node.DataContract()
		writes := make(map[string]cnst.MFormat)
		for _, w := range contract.Writes {
			if !strings.HasPrefix(w, cnst.RuleMsgPrefix) {
				continue
			}
			if err := asset.ValidateURI(w); err != nil {
				continue
			}
			path, format, ok := parseDataURI(w)
			if ok {
				// We only care about data writes for strict format checking
				// Use path as key (usually empty string for full data write)
				writes[path] = format
			}
		}
		nodeWrites[id] = writes
	}

	// 2. Check reads against upstream writes
	// Traverse connections to find upstream nodes
	// Since graph can be complex, we do a simple check: for each node, check its reads against direct predecessors
	// Note: This is a simplified check. Real data flow analysis needs full graph traversal.
	// But Matrix Execute passes full msg, so direct predecessor write is visible to successor.

	// Invert connections map for easy lookup of predecessors
	predecessors := make(map[string][]string)
	for _, conn := range instance.Definition().Metadata.Connections {
		predecessors[conn.ToID] = append(predecessors[conn.ToID], conn.FromID)
	}

	for id, node := range nodes {
		contract := node.DataContract()
		for _, r := range contract.Reads {
			if !strings.HasPrefix(r, cnst.RuleMsgPrefix) {
				continue
			}
			if err := asset.ValidateURI(r); err != nil {
				continue
			}
			path, readFormat, ok := parseDataURI(r)
			if !ok {
				continue // Not a data URI or invalid
			}

			// Check all predecessors
			preds := predecessors[id]
			if len(preds) == 0 {
				// Root node reading data? Ideally should come from external input (msg).
				// We can't validate external input staticly easily without more context.
				continue
			}

			// For now, strict rule: At least one predecessor MUST write this data with compatible format.
			// Or should it be ALL? In a branching flow, maybe only one path is taken.
			// Let's enforce: If a predecessor writes 'data', it MUST match the read format.
			// If no predecessor writes 'data', it might be from earlier upstream or initial input (pass).

			for _, predID := range preds {
				predWrites := nodeWrites[predID]
				if writeFormat, written := predWrites[path]; written {
					if writeFormat != readFormat {
						return fmt.Errorf("node '%s' reads data (format %s) but upstream node '%s' writes format %s",
							node.Name(), readFormat, instance.Definition().Metadata.Nodes[findNodeIndex(instance.Definition().Metadata.Nodes, predID)].Name, writeFormat)
					}
				}
			}
		}
	}

	return nil
}

func findNodeIndex(nodes []types.NodeDef, id string) int {
	for i, n := range nodes {
		if n.ID == id {
			return i
		}
	}
	return -1
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
	r.coreObjPlan = rulechain.AnalyzeCoreObjProjection(newChainDef, newInstance)
	r.mutex.Unlock()

	// 3. Destroy the old instance after the swap is complete.
	//    This ensures that new executions use the new instance, while
	//    ongoing executions on the old instance can finish gracefully.
	if oldInstance != nil {
		oldInstance.Destroy()
	}

	return nil
}

// Destroy releases all resources held by the runtime.
func (r *DefaultRuntime) Destroy() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.chainInstance != nil {
		r.chainInstance.Destroy()
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
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.nodePool
}

// GetEngine returns the engine instance associated with this runtime.
func (r *DefaultRuntime) GetEngine() types.MatrixEngine {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.engine
}

// GetChainInstance returns the chain instance for accessing initialized nodes.
func (r *DefaultRuntime) GetChainInstance() types.ChainInstance {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.chainInstance
}

// CoreObjProjection returns the cached rulechain projection plan.
func (r *DefaultRuntime) CoreObjProjection() types.RuleChainCoreObjAnalysis {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.coreObjPlan
}

// LiveObjectsForEdge returns the cached live-object set for a concrete execution edge.
func (r *DefaultRuntime) LiveObjectsForEdge(fromNodeID string, toNodeID string) (types.CoreObjSet, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	if r.coreObjPlan.LiveObjectsByEdge == nil {
		return types.CoreObjSet{}, false
	}
	set, ok := r.coreObjPlan.LiveObjectsByEdge[rulechain.LiveObjectsEdgeKey(fromNodeID, toNodeID)]
	return set, ok
}
