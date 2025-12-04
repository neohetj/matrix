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
	"sync/atomic"
	"time"

	"gitlab.com/neohet/matrix/internal/log"
	"gitlab.com/neohet/matrix/pkg/types"
)

// ChainInstance holds the live state of a running rule chain execution.
type ChainInstance struct {
	def         *types.RuleChainDef
	nodes       map[string]types.Node
	nodeDefs    map[string]*types.NodeDef
	connections map[string][]types.Connection
}

// DefaultNodeCtx is the default implementation of the NodeCtx interface.
type DefaultNodeCtx struct {
	context.Context
	config       types.Config
	selfDef      *types.NodeDef
	runtime      *DefaultRuntime
	chain        *ChainInstance
	onEnd        func(msg types.RuleMsg, err error)
	parentCtx    *DefaultNodeCtx
	waitingCount int32
	aspects      []types.Aspect
	callback     types.CallbackFunc
}

// NewDefaultNodeCtx creates a new node context.
func NewDefaultNodeCtx(ctx context.Context, r *DefaultRuntime, chain *ChainInstance, selfDef *types.NodeDef, parent *DefaultNodeCtx, onEnd func(msg types.RuleMsg, err error), aspects []types.Aspect, callback types.CallbackFunc) *DefaultNodeCtx {
	return &DefaultNodeCtx{
		Context:   ctx,
		runtime:   r,
		chain:     chain,
		selfDef:   selfDef,
		config:    selfDef.Configuration,
		parentCtx: parent,
		onEnd:     onEnd,
		aspects:   aspects,
		callback:  callback,
	}
}

// childReady increases the waiting counter of the parent context.
func (ctx *DefaultNodeCtx) childReady() {
	atomic.AddInt32(&ctx.waitingCount, 1)
}

// childDone decreases the waiting counter and triggers the onEnd callback if it reaches zero.
func (ctx *DefaultNodeCtx) childDone(msg types.RuleMsg, err error) {
	// Node-level completion callback, skip for the virtual root context.
	if ctx.callback != nil && ctx.selfDef.ID != "root" {
		ctx.callback.OnNodeCompleted(ctx, msg, err)
	}

	if atomic.AddInt32(&ctx.waitingCount, -1) <= 0 {
		if ctx.parentCtx != nil {
			// Propagate completion to the parent
			ctx.parentCtx.childDone(msg, err)
		} else if ctx.onEnd != nil {
			// This is the root context, and all branches are done
			ctx.onEnd(msg, err)
		}
	}
}

// GetContext returns the context.Context.
func (ctx *DefaultNodeCtx) GetContext() context.Context {
	return ctx.Context
}

// SetContext sets the context.Context.
func (ctx *DefaultNodeCtx) SetContext(c context.Context) {
	ctx.Context = c
}

// Config returns the node's configuration.
func (ctx *DefaultNodeCtx) Config() types.Config {
	return ctx.config
}

// ChainConfig returns the configuration of the current rule chain.
func (ctx *DefaultNodeCtx) ChainConfig() types.Config {
	if ctx.chain == nil || ctx.chain.def == nil {
		return nil
	}
	return ctx.chain.def.RuleChain.Configuration
}

// ChainID returns the ID of the current rule chain.
func (ctx *DefaultNodeCtx) ChainID() string {
	if ctx.chain == nil || ctx.chain.def == nil {
		return ""
	}
	return ctx.chain.def.RuleChain.ID
}

// Logger returns the logger instance for the current runtime context.
func (ctx *DefaultNodeCtx) Logger() types.Logger {
	if ctx.runtime == nil {
		return nil
	}
	return ctx.runtime.logger
}

// SelfDef returns the definition of the current node.
func (ctx *DefaultNodeCtx) SelfDef() *types.NodeDef {
	return ctx.selfDef
}

// NodeID returns the node id.
func (ctx *DefaultNodeCtx) NodeID() string {
	if ctx.selfDef == nil {
		return ""
	}
	return ctx.selfDef.ID
}

// GetNodeById checks if a node with the given ID exists in the current rule chain.
func (ctx *DefaultNodeCtx) GetNodeById(id string) bool {
	if ctx.chain == nil || ctx.chain.nodes == nil {
		return false
	}
	_, ok := ctx.chain.nodes[id]
	return ok
}

// TellSuccess finds the "Success" relation and calls TellNext.
func (ctx *DefaultNodeCtx) TellSuccess(msg types.RuleMsg) {
	ctx.TellNext(msg, "Success")
}

// TellFailure routes the message to the 'Failure' relation.
// It is a low-level routing method. For standard error handling,
// developers should use HandleError, which calls this method internally.
func (ctx *DefaultNodeCtx) TellFailure(msg types.RuleMsg, err error) {
	// Log the failure routing event for better traceability.
	// The main error is logged by HandleError, so this is just an info-level log.
	if def := ctx.SelfDef(); def != nil {
		ctx.Info("Routing message to 'Failure' output due to error",
			"nodeId", def.ID,
			"nodeName", def.Name,
			"nodeType", def.Type,
			"error", err)
	} else {
		ctx.Info("Routing message to 'Failure' output due to error", "error", err)
	}

	// The error is logged and added to metadata by HandleError.
	// This method's responsibility is to route the message to the failure path.
	// The original error is passed to childDone to be available for AOP aspects.
	ctx.TellNext(msg, "Failure")
}

// HandleError provides a standardized way to process errors within a node.
// It logs the error and then routes the message to the 'Failure' output.
func (ctx *DefaultNodeCtx) HandleError(msg types.RuleMsg, err error) {
	// 1. Log the error with structured context.
	ctx.Error("Node execution failed", "error", err)

	// 2. Enrich the message metadata.
	if msg.Metadata() == nil {
		msg.SetMetadata(make(types.Metadata))
	}
	metadata := msg.Metadata()
	metadata[types.MetaError] = err.Error()
	if ctx.SelfDef() != nil {
		metadata[types.MetaErrorNodeID] = ctx.SelfDef().ID
		metadata[types.MetaErrorNodeName] = ctx.SelfDef().Name
	}
	metadata[types.MetaErrorTimestamp] = time.Now().UTC().Format(time.RFC3339)

	var fault *types.Fault
	if errors.As(err, &fault) {
		metadata[types.MetaErrorCode] = fmt.Sprintf("%d", fault.Code)
	}

	// 3. Route the message to the failure path.
	ctx.TellFailure(msg, err)
}

// TellNext finds the next nodes based on relation types and submits them for execution.
// If no next node is found for any of the given relation types, it signals its own completion.
func (ctx *DefaultNodeCtx) TellNext(msg types.RuleMsg, relationTypes ...string) {
	if ctx.chain == nil || ctx.selfDef == nil {
		ctx.childDone(msg, nil)
		return
	}

	// Get all connections for the current node
	connections, ok := ctx.chain.connections[ctx.selfDef.ID]
	if !ok {
		// No connections from this node, so it's a leaf
		ctx.childDone(msg, nil)
		return
	}

	foundNext := false
	for _, relationType := range relationTypes {
		for _, conn := range connections {
			if conn.Type == relationType {
				nextNodeID := conn.ToID
				if nextNode, nodeOk := ctx.chain.nodes[nextNodeID]; nodeOk {
					if nextDef, defOk := ctx.chain.nodeDefs[nextNodeID]; defOk {
						foundNext = true
						// Increment parent's waiting counter before submitting the task
						ctx.childReady()

						// Create a new context for the child node
						// The child's onEnd is the parent's childDone
						nextCtx := NewDefaultNodeCtx(ctx.Context, ctx.runtime, ctx.chain, nextDef, ctx, ctx.onEnd, ctx.aspects, ctx.callback)

						// Create a copy of the message for each parallel branch.
						// This isolates metadata while sharing the main data object (DataT).
						msgCopy := msg.Copy()

						ctx.runtime.scheduler.Submit(func() {
							var onMsgErr error
							var processedMsg = msgCopy

							// Defer the After aspect calls.
							// It captures the final state of processedMsg and onMsgErr.
							defer func() {
								for _, aspect := range nextCtx.aspects {
									aspect.After(nextCtx, processedMsg, onMsgErr)
								}
							}()

							// 1. Execute Before aspects
							for _, aspect := range nextCtx.aspects {
								var err error
								processedMsg, err = aspect.Before(nextCtx, processedMsg)
								if err != nil {
									onMsgErr = err
									// If Before aspect fails, skip OnMsg and signal completion immediately
									nextCtx.childDone(processedMsg, onMsgErr)
									return
								}
							}

							// 2. Execute the node's OnMsg
							// The node itself is responsible for calling childDone via Tell... methods
							nextNode.OnMsg(nextCtx, processedMsg)
						})
					}
				}
			}
		}
	}

	// If no subsequent nodes were found for the given relation types, this path is complete.
	if !foundNext {
		ctx.childDone(msg, nil)
	}
}

// NewMsg creates a new message with a new message ID.
func (ctx *DefaultNodeCtx) NewMsg(msgType string, metaData types.Metadata, data string) types.RuleMsg {
	// By default, messages created within a chain are treated as TEXT, unless specified otherwise.
	return types.NewMsg(msgType, data, metaData, nil).WithDataFormat("TEXT")
}

// SetOnAllNodesCompleted sets a callback that will be called when all nodes in the chain have completed.
// TODO: Implement this properly.
func (ctx *DefaultNodeCtx) SetOnAllNodesCompleted(f func()) {
}

// GetRuntime returns the runtime instance associated with this context.
func (ctx *DefaultNodeCtx) GetRuntime() types.Runtime {
	return ctx.runtime
}

// --- Logging Methods ---

// logWithFields is a private helper that prepares a logger with contextual fields.
func (ctx *DefaultNodeCtx) logWithFields(fields ...any) types.Logger {
	logger := ctx.Logger()
	if logger == nil {
		// Fallback to the global logger if the context-specific one isn't available.
		logger = log.GetLogger()
	}

	// Extract base fields from the context.
	var baseFields []any
	if chainID := ctx.ChainID(); chainID != "" {
		baseFields = append(baseFields, "chainId", chainID)
	}
	if nodeID := ctx.NodeID(); nodeID != "" {
		baseFields = append(baseFields, "nodeId", nodeID)
	}

	// Merge base fields with the provided business fields.
	allFields := append(baseFields, fields...)

	// Use With to attach all fields and return.
	return logger.With(allFields...)
}

// Debug logs a message at Debug level with context-aware fields.
func (ctx *DefaultNodeCtx) Debug(msg string, fields ...any) {
	ctx.logWithFields(fields...).Debugf(ctx.GetContext(), msg)
}

// Info logs a message at Info level with context-aware fields.
func (ctx *DefaultNodeCtx) Info(msg string, fields ...any) {
	ctx.logWithFields(fields...).Infof(ctx.GetContext(), msg)
}

// Warn logs a message at Warn level with context-aware fields.
func (ctx *DefaultNodeCtx) Warn(msg string, fields ...any) {
	ctx.logWithFields(fields...).Warnf(ctx.GetContext(), msg)
}

// Error logs a message at Error level with context-aware fields.
func (ctx *DefaultNodeCtx) Error(msg string, fields ...any) {
	ctx.logWithFields(fields...).Errorf(ctx.GetContext(), msg)
}
