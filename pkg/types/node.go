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

// Package types defines the core data structures and interfaces for the Matrix rule engine.

package types

import (
	"context"
)

// MetadataDef 描述了对一个元数据键的读或写
type MetadataDef struct {
	Key         string `json:"key"`
	Description string `json:"description"`
}

// DataContract 包含了节点所有的数据访问契约
type DataContract struct {
	ReadsData      []string
	ReadsMetadata  []MetadataDef
	WritesMetadata []MetadataDef
	Inputs         []IOObject // For DataT
	Outputs        []IOObject // For DataT
}

// NodeDefinition contains static, descriptive metadata about a Node component.
// This information is used for discovery, documentation, and UI rendering.
type NodeDefinition struct {
	Type        string   `json:"type"`           // The unique, machine-readable type of the node.
	Name        string   `json:"name"`           // The human-readable name.
	Description string   `json:"description"`    // e.g., "Prints a message to the console."
	Dimension   string   `json:"dimension"`      // e.g., "System", "Business Logic", "Integration"
	Tags        []string `json:"tags"`           // e.g., ["action", "debug"]
	Version     string   `json:"version"`        // The semantic version of the node component.
	Icon        string   `json:"icon,omitempty"` // A string identifier for the node's icon in a UI.

	// ReadsData (可选) 声明节点从原始 RuleMsg.Data 中读取的字段路径列表。
	// 例如: ["customer.name", "order.id"]
	ReadsData []string `json:"readsData,omitempty"`
	// ReadsMetadata (可选) 声明节点读取的元数据键。
	ReadsMetadata []MetadataDef `json:"readsMetadata,omitempty"`
	// WritesMetadata (可选) 声明节点写入的元数据键。
	WritesMetadata []MetadataDef `json:"writesMetadata,omitempty"`
}

// DynamicConfigField defines a single dynamic configuration field for a component.
// This is used for self-description, allowing for dynamic UI generation and validation.
type DynamicConfigField struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Desc     string `json:"description"`
	Required bool   `json:"required"`
	Default  any    `json:"defaultValue,omitempty"`
}

// RuleContext is the interface for the context of a message processing.
type RuleContext interface {
	GetContext() context.Context
	SetContext(ctx context.Context)
	ChainConfig() Config
	ChainID() string
	Logger() Logger
}

// NodeCtxLogger defines the logging methods available on a NodeCtx.
type NodeCtxLogger interface {
	Debug(msg string, fields ...any)
	Info(msg string, fields ...any)
	Warn(msg string, fields ...any)
	Error(msg string, fields ...any)
}

// NodeCtx is the context for a node.
type NodeCtx interface {
	RuleContext
	NodeCtxLogger
	NodeID() string
	GetNodeById(id string) bool
	TellSuccess(msg RuleMsg)
	TellFailure(msg RuleMsg, err error)
	HandleError(msg RuleMsg, err error)
	TellNext(msg RuleMsg, relationTypes ...string)
	NewMsg(msgType string, metaData Metadata, data string) RuleMsg
	Config() Config
	SelfDef() *NodeDef
	GetRuntime() Runtime
	SetOnAllNodesCompleted(f func())
}

// NodeType is the type of a node.
type NodeType string

// Node is the base interface for all components in the rule chain.
type Node interface {
	// New returns a new, uninitialized instance of the node.
	// This is called by the NodeManager when creating a node instance for a rule chain.
	New() Node
	// Type returns the static type of the node, e.g., "action/log".
	// This should be implemented by the concrete node struct, not the BaseNode,
	// to ensure the prototype registered in init() can provide its type.
	Type() NodeType
	// Init initializes the node instance with its specific configuration from the DSL.
	Init(configuration Config) error
	// OnMsg is the message processing function of the node.
	OnMsg(ctx NodeCtx, msg RuleMsg)
	// Destroy is called when the node is no longer needed, allowing it to clean up resources.
	Destroy()
	// Definition returns the static metadata (template) of the node type.
	Definition() NodeDefinition
	// GetDataContract returns a unified view of the node's data access contract.
	GetDataContract() DataContract
	// ID returns the unique identifier of this specific node instance, as defined in the DSL.
	ID() string
	// Name returns the name of this specific node instance.
	Name() string

	// SetID sets the unique identifier for this node instance.
	// This is called by the runtime during initialization.
	SetID(id string)
	// SetName sets the name for this node instance.
	// This is called by the runtime during initialization.
	SetName(name string)
}

// Instance holds instance-specific metadata for a node.
// It can be embedded into concrete node structs to provide
// standard implementations for ID(), Name(), SetID(), and SetName().
type Instance struct {
	id   string
	name string
}

// ID returns the instance-specific ID.
func (i *Instance) ID() string {
	return i.id
}

// Name returns the instance-specific name.
func (i *Instance) Name() string {
	return i.name
}

// SetID sets the instance-specific ID.
func (i *Instance) SetID(id string) {
	i.id = id
}

// SetName sets the instance-specific name.
func (i *Instance) SetName(name string) {
	i.name = name
}

// PassiveEndpoint is a marker interface for Endpoints that are triggered by external services.
type PassiveEndpoint interface {
	Endpoint
}

// ActiveEndpoint is an interface for Endpoints that actively listen for events.
type ActiveEndpoint interface {
	Endpoint
	Start(ctx context.Context) error
	Stop() error
}

// Endpoint is a special type of Node that acts as an entry point to a rule chain.
// By design, Endpoints are also considered SharedNodes, as they are typically instantiated
// once at the application level and their lifecycle is not tied to a single rule chain execution.
type Endpoint interface {
	Node
	SharedNode // Endpoints must be shareable.
	// SetRuntimePool allows an external manager to inject the runtime pool dependency.
	// The pool instance is passed as an any to avoid import cycles.
	SetRuntimePool(pool any) error
}

// SharedNode represents a node component that manages a shareable resource.
type SharedNode interface {
	Node
	GetInstance() (any, error)
}

// BaseNode holds static, shared information for a node type.
// It acts as a template and is shared by all instances of a given node type.
type BaseNode struct {
	def NodeDefinition
}

// NewBaseNode creates a new BaseNode template.
func NewBaseNode(nodeType NodeType, def NodeDefinition) *BaseNode {
	def.Type = string(nodeType)
	return &BaseNode{def: def}
}

// Type returns the type of the node from the template.
func (n *BaseNode) Type() NodeType {
	return NodeType(n.def.Type)
}

// Definition returns the static metadata of the node from the template.
func (n *BaseNode) Definition() NodeDefinition {
	return n.def
}

// OnMsg provides a default no-op implementation.
func (n *BaseNode) OnMsg(ctx NodeCtx, msg RuleMsg) { ctx.TellSuccess(msg) }

// Destroy provides a default no-op implementation.
func (n *BaseNode) Destroy() {}

// GetDataContract provides a default implementation for non-function nodes.
// It builds the contract from the fields in the NodeDefinition.
func (n *BaseNode) GetDataContract() DataContract {
	return DataContract{
		ReadsData:      n.def.ReadsData,
		ReadsMetadata:  n.def.ReadsMetadata,
		WritesMetadata: n.def.WritesMetadata,
		Inputs:         nil, // Base nodes don't have DataT inputs
		Outputs:        nil, // Base nodes don't have DataT outputs
	}
}
