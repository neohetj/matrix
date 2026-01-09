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

package registry

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/neohetj/matrix/internal/log"
	"github.com/neohetj/matrix/internal/parser"
	"github.com/neohetj/matrix/pkg/types"
)

var (
	ErrNotSharedNode = errors.New("node does not implement types.SharedNode interface")
)

// nodePool is the default implementation of the types.NodePool interface.
// It uses a sync.Map to store and manage shared node instances.
type nodePool struct {
	// entries stores the shared nodes, with the node ID as the key.
	entries sync.Map
	// parser is used to decode DSLs.
	parser types.Parser
	// endpoints holds a dedicated list of nodes that implement the Endpoint interface.
	endpoints []types.Endpoint
}

// NewNodePool creates a new NodePool.
func NewNodePool(p types.Parser) types.NodePool {
	if p == nil {
		p = &parser.JsonParser{}
	}
	return &nodePool{
		parser:    p,
		endpoints: make([]types.Endpoint, 0),
	}
}

// Load loads a list of shared nodes from a rule chain DSL definition.
func (p *nodePool) Load(dsl []byte, nodeMgr types.NodeManager) (types.NodePool, error) {
	def, err := p.parser.DecodeRuleChain(dsl)
	if err != nil {
		return nil, err
	}
	return p.LoadFromRuleChainDef(def, nodeMgr)
}

// LoadFromRuleChainDef loads a list of shared nodes from a rule chain definition.
func (p *nodePool) LoadFromRuleChainDef(def *types.RuleChainDef, nodeMgr types.NodeManager) (types.NodePool, error) {
	for _, nodeDef := range def.Metadata.Nodes {
		if _, err := p.NewFromNodeDef(nodeDef, nodeMgr); err != nil {
			return nil, err
		}
	}
	return p, nil
}

// NewFromNodeDef creates and adds a new shared node from a node definition.
func (p *nodePool) NewFromNodeDef(def types.NodeDef, nodeMgr types.NodeManager) (types.SharedNodeCtx, error) {
	if _, ok := p.entries.Load(def.ID); ok {
		return nil, fmt.Errorf("duplicate shared node id: %s", def.ID)
	}

	node, err := nodeMgr.NewNode(types.NodeType(def.Type))
	if err != nil {
		return nil, err
	}

	// V2 Initialization Flow:
	// 1. Set instance-specific metadata.
	node.SetID(def.ID)
	node.SetName(def.Name)

	// 2. Initialize with business-specific configuration only.
	if err := node.Init(def.Configuration); err != nil {
		return nil, fmt.Errorf("failed to initialize node '%s': %w", def.ID, err)
	}

	sharedNode, ok := node.(types.SharedNode)
	if !ok {
		return nil, fmt.Errorf("node type=%s, id=%s: %w", def.Type, def.ID, ErrNotSharedNode)
	}

	// If the node is also an endpoint, add it to the dedicated list.
	if endpoint, ok := node.(types.Endpoint); ok {
		p.AddEndpoint(endpoint)
	}

	// Create a minimal NodeCtx for the shared node.
	// This context is primarily for initialization and holding configuration.
	nodeCtx := &minimalNodeCtx{
		nodeDef: def,
	}

	sharedCtx := &defaultSharedNodeCtx{
		NodeCtx:    nodeCtx,
		sharedNode: sharedNode,
	}

	p.entries.Store(def.ID, sharedCtx)
	return sharedCtx, nil
}

// Get retrieves a shared node context by its ID.
func (p *nodePool) Get(id string) (types.SharedNodeCtx, bool) {
	v, ok := p.entries.Load(id)
	if !ok {
		return nil, false
	}
	return v.(types.SharedNodeCtx), true
}

// GetInstance retrieves a shared resource instance by its node ID.
func (p *nodePool) GetInstance(id string) (any, error) {
	ctx, ok := p.Get(id)
	if !ok {
		return nil, fmt.Errorf("shared node resource not found, id=%s", id)
	}
	return ctx.GetInstance()
}

// Del removes a shared node from the pool and destroys it.
func (p *nodePool) Del(id string) {
	if v, loaded := p.entries.LoadAndDelete(id); loaded {
		v.(types.SharedNodeCtx).GetNode().Destroy()
	}
}

// Stop stops and releases all shared node instances in the pool.
func (p *nodePool) Stop() {
	p.entries.Range(func(key, value any) bool {
		p.Del(key.(string))
		return true
	})
}

// GetAll returns all shared node contexts in the pool.
func (p *nodePool) GetAll() []types.NodeCtx {
	var items []types.NodeCtx
	p.entries.Range(func(key, value any) bool {
		items = append(items, value.(types.SharedNodeCtx))
		return true
	})
	return items
}

// AddEndpoint adds a node instance identified as an endpoint to a dedicated internal list.
func (p *nodePool) AddEndpoint(endpoint types.Endpoint) {
	p.endpoints = append(p.endpoints, endpoint)
}

// GetEndpoints returns all nodes that have been identified and added as endpoints.
func (p *nodePool) GetEndpoints() []types.Endpoint {
	return p.endpoints
}

// defaultSharedNodeCtx is the default implementation of SharedNodeCtx.
type defaultSharedNodeCtx struct {
	types.NodeCtx
	sharedNode types.SharedNode
}

// GetInstance obtains the shared resource instance from the node.
func (c *defaultSharedNodeCtx) GetInstance() (interface{}, error) {
	return c.sharedNode.GetInstance()
}

// GetNode returns the underlying node instance.
func (c *defaultSharedNodeCtx) GetNode() types.Node {
	return c.sharedNode
}

// minimalNodeCtx is a lightweight, private implementation of types.NodeCtx.
// It's used for shared nodes within the pool, providing just enough context
// for initialization and configuration access, without depending on the runtime package.
type minimalNodeCtx struct {
	nodeDef types.NodeDef
}

// NewMinimalNodeCtx creates a new MinimalNodeCtx for a given node ID.
// This is useful for logging in contexts where a full NodeCtx is not available,
// such as within the shared node's own methods.
func NewMinimalNodeCtx(nodeId string) types.NodeCtx {
	return &minimalNodeCtx{
		nodeDef: types.NodeDef{
			ID: nodeId,
		},
	}
}

func (m *minimalNodeCtx) GetContext() context.Context              { return context.Background() }
func (m *minimalNodeCtx) SetContext(ctx context.Context)           {}
func (m *minimalNodeCtx) ChainConfig() types.ConfigMap             { return nil }
func (m *minimalNodeCtx) ChainID() string                          { return "" } // Not part of a chain
func (m *minimalNodeCtx) Logger() types.Logger                     { return log.GetLogger() }
func (m *minimalNodeCtx) NodeID() string                           { return m.nodeDef.ID }
func (m *minimalNodeCtx) GetNode() types.Node                      { return nil }
func (m *minimalNodeCtx) TellSuccess(msg types.RuleMsg)            {}
func (m *minimalNodeCtx) TellFailure(msg types.RuleMsg, err error) {}
func (m *minimalNodeCtx) HandleError(msg types.RuleMsg, err error) {
	// minimalNodeCtx does not have a failure path, so it only logs the error.
	m.Error("Node execution failed in minimal context", "error", err)
}
func (m *minimalNodeCtx) TellNext(msg types.RuleMsg, relationTypes ...string) {}
func (m *minimalNodeCtx) NewMsg(msgType string, metaData types.Metadata, data string) types.RuleMsg {
	return nil
}
func (m *minimalNodeCtx) Config() types.ConfigMap { return m.nodeDef.Configuration }
func (m *minimalNodeCtx) SelfDef() *types.NodeDef {
	return &m.nodeDef
}
func (m *minimalNodeCtx) GetRuntime() types.Runtime {
	return nil // minimalNodeCtx is not associated with a runtime.
}
func (m *minimalNodeCtx) SetOnAllNodesCompleted(f func()) {}

// --- Logging Methods (No-op for minimal context) ---

func (m *minimalNodeCtx) Debug(msg string, fields ...interface{}) {}
func (m *minimalNodeCtx) Info(msg string, fields ...interface{})  {}
func (m *minimalNodeCtx) Warn(msg string, fields ...interface{})  {}
func (m *minimalNodeCtx) Error(msg string, fields ...interface{}) {
	// For minimal context, we can log the error to the global logger as a fallback.
	// This is better than silently swallowing it.
	logger := m.Logger()
	if logger != nil {
		allFields := append([]interface{}{"nodeId", m.NodeID()}, fields...)
		logger.With(allFields...).Errorf(context.Background(), msg)
	}
}
