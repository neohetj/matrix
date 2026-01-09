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

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/neohetj/matrix/pkg/cnst"
)

// ErrorMapping defines a map from an external protocol code/status (string)
// to a list of internal Fault codes (string).
// Key: Protocol-specific error code (e.g., "404", "500" for HTTP; "RETRY", "DLQ" for Streams).
// Value: A list of internal Fault.Code values that map to this protocol code.
type ErrorMapping map[string][]string

// EndpointIOField defines the mapping for a single field between an external protocol (e.g., HTTP) and a RuleMsg.
type EndpointIOField struct {
	// Name is the field name in the external protocol (e.g., HTTP Header key, Query param name, JSON field name).
	Name string `json:"name"`
	// BindPath is the path in the RuleMsg to bind this field to (e.g., "rulemsg://metadata.token", "rulemsg://dataT.user.id?sid=xxx").
	BindPath string `json:"bindPath"`
	// Type specifies the data type for conversion.
	Type cnst.MType `json:"type"`
	// Required indicates if the field must be present.
	Required bool `json:"required,omitempty"`
	// DefaultValue is used if the field is missing.
	DefaultValue any `json:"defaultValue,omitempty"`
	// Description describes the field.
	Description string `json:"description,omitempty"`
}

// EndpointIOPacket defines the mapping for a group of data (e.g., HTTP Body, Headers).
// It supports mapping the entire packet to a single path (MapAll) OR mapping individual fields (Fields).
type EndpointIOPacket struct {
	// MapAll, if set, maps the entire data packet (e.g., whole Body or all Headers) to/from this path in RuleMsg.
	MapAll *string `json:"mapAll,omitempty"`

	// Fields defines mappings for individual fields within the packet.
	// Can be used in conjunction with MapAll to override or supplement specific fields.
	Fields []EndpointIOField `json:"fields,omitempty"`
}

// ContractDef defines a single data access requirement with a URI and description.
// URI format:
// - rulemsg://data/xxx.xxx (accessing RuleMsg.Data)
// - rulemsg://metadata/xxx.xxx (accessing RuleMsg.Metadata)
// - rulemsg://dataT/ObjId.xxx (accessing DataT objects)
type ContractDef struct {
	URI         string `json:"uri"`
	Description string `json:"description"`
}

// DataContract 包含了节点所有的数据访问契约
type DataContract struct {
	Reads  []string
	Writes []string
}

// NodeMetadata contains static, descriptive metadata about a Node component.
// This information is used for discovery, documentation, and UI rendering.
type NodeMetadata struct {
	Type        string   `json:"type"`           // The unique, machine-readable type of the node.
	Name        string   `json:"name"`           // The human-readable name.
	Description string   `json:"description"`    // e.g., "Prints a message to the console."
	Dimension   string   `json:"dimension"`      // e.g., "System", "Business Logic", "Integration"
	Tags        []string `json:"tags"`           // e.g., ["action", "debug"]
	Version     string   `json:"version"`        // The semantic version of the node component.
	Icon        string   `json:"icon,omitempty"` // A string identifier for the node's icon in a UI.

	// NodeReads declares the static data requirements of the node.
	NodeReads []ContractDef `json:"nodeReads,omitempty"`
	// NodeWrites declares the static data production of the node.
	NodeWrites []ContractDef `json:"nodeWrites,omitempty"`
}

// DynamicConfigField defines a single dynamic configuration field for a component.
// This is used for self-description, allowing for dynamic UI generation and validation.
type DynamicConfigField struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Type     cnst.MType `json:"type"`
	Desc     string     `json:"description"`
	Required bool       `json:"required"`
	Default  any        `json:"defaultValue,omitempty"`
}

// RuleContext is the interface for the context of a message processing.
type RuleContext interface {
	GetContext() context.Context
	SetContext(ctx context.Context)
	ChainConfig() ConfigMap
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

	// NodeID returns the unique identifier of the current node.
	NodeID() string

	// GetNode returns the current Node instance from Chain Instance.
	GetNode() Node

	// TellSuccess is a convenience method to route the message to the 'Success' relation.
	// It is equivalent to calling TellNext(msg, "Success").
	TellSuccess(msg RuleMsg)

	// TellFailure routes the message to the 'Failure' relation.
	// It is a low-level routing method. For standard error handling, use HandleError.
	TellFailure(msg RuleMsg, err error)

	// HandleError provides a standardized way to process errors within a node.
	// It logs the error, enriches the message metadata, and routes it to the 'Failure' output.
	HandleError(msg RuleMsg, err error)

	// TellNext routes the message to the next nodes connected via the specified relation types.
	TellNext(msg RuleMsg, relationTypes ...string)

	// NewMsg creates a new RuleMsg with the specified type, metadata, and data.
	NewMsg(msgType string, metaData Metadata, data string) RuleMsg

	// Config returns the configuration map for the current node.
	Config() ConfigMap

	// SelfDef returns the definition (NodeDef) of the current node.
	SelfDef() *NodeDef

	// GetRuntime returns the runtime instance managing this node context.
	GetRuntime() Runtime

	// SetOnAllNodesCompleted sets a callback function to be executed when all nodes in the chain have completed.
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
	Init(configuration ConfigMap) error
	// OnMsg is the message processing function of the node.
	OnMsg(ctx NodeCtx, msg RuleMsg)
	// Destroy is called when the node is no longer needed, allowing it to clean up resources.
	Destroy()
	// NodeMetadata returns the static metadata (template) of the node type.
	NodeMetadata() NodeMetadata
	// GetDataContract returns a unified view of the node's data access contract.
	DataContract() DataContract
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

	// Errors returns the list of possible faults that this node can produce.
	Errors() []*Fault
	// ConfigSchema returns the OpenAPI 3.0 schema for the node's configuration.
	ConfigSchema() *openapi3.Schema
}

// NodeDefBinding allows a node to receive its rule-chain definition for binding/mapping.
type NodeDefBinding interface {
	BindNodeDef(def *NodeDef)
	GetNodeDef() *NodeDef
}

// SubChainTrigger is an interface for nodes that execute a sub-chain.
// It exposes the input and output mappings to allow external tools (like UI)
// to visualize and manage the data flow between the parent and the sub-chain.
//
// Role Separation:
//   - Node.DataContract(): Specifies what the trigger node itself needs from the PARENT chain.
//   - SubChainTrigger (Mappings): Specifies what the SUB-CHAIN will receive (InputMapping) and what it must produce (OutputMapping).
//     This effectively defines the "Dynamic Data Contract" for the sub-chain, eliminating the need for a separate contract interface.
type SubChainTrigger interface {
	Node
	// GetInputMapping returns the configuration for mapping data from the parent message to the sub-chain message.
	GetInputMapping() EndpointIOPacket
	// GetOutputMapping returns the configuration for mapping data from the sub-chain result back to the parent message.
	GetOutputMapping() EndpointIOPacket
	// GetTargetChainID returns the ID of the sub-chain being triggered.
	GetTargetChainID() string
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

// BaseNode holds static, shared information for a node type.
// It acts as a template and is shared by all instances of a given node type.
type BaseNode struct {
	nodeMeta NodeMetadata
	nodeDef  *NodeDef
}

// NewBaseNode creates a new BaseNode template.
func NewBaseNode(nodeType NodeType, def NodeMetadata) *BaseNode {
	def.Type = string(nodeType)
	return &BaseNode{nodeMeta: def}
}

// Type returns the type of the node from the template.
func (n *BaseNode) Type() NodeType {
	return NodeType(n.nodeMeta.Type)
}

// BindNodeDef stores the rule-chain node definition on the instance.
func (n *BaseNode) BindNodeDef(def *NodeDef) {
	n.nodeDef = def
}

// GetNodeDef returns the bound rule-chain node definition.
func (n *BaseNode) GetNodeDef() *NodeDef {
	return n.nodeDef
}

// NodeMetadata returns the static metadata of the node from the template.
func (n *BaseNode) NodeMetadata() NodeMetadata {
	return n.nodeMeta
}

// OnMsg provides a default no-op implementation.
func (n *BaseNode) OnMsg(ctx NodeCtx, msg RuleMsg) { ctx.TellSuccess(msg) }

// Destroy provides a default no-op implementation.
func (n *BaseNode) Destroy() {}

// GetDataContract provides a default implementation for non-function nodes.
// It builds the contract from the fields in the NodeMetadata.
func (n *BaseNode) DataContract() DataContract {
	contract := DataContract{
		Reads:  make([]string, len(n.nodeMeta.NodeReads)),
		Writes: make([]string, len(n.nodeMeta.NodeWrites)),
	}

	for i, r := range n.nodeMeta.NodeReads {
		contract.Reads[i] = r.URI
	}
	for i, w := range n.nodeMeta.NodeWrites {
		contract.Writes[i] = w.URI
	}

	return contract
}

// Errors provides a default implementation returning nil.
func (n *BaseNode) Errors() []*Fault {
	return nil
}

// ConfigSchema provides a default implementation returning nil.
func (n *BaseNode) ConfigSchema() *openapi3.Schema {
	return nil
}

// NodeFunc is the function signature for a node's logic.
// It receives a node-level context, allowing it to control routing.
type NodeFunc func(ctx NodeCtx, msg RuleMsg)

// IOObject defines the metadata for an input or output parameter of a function node.
type IOObject struct {
	ParamName string `json:"paramName"`
	DefineSID string `json:"defineSid"`
	Desc      string `json:"desc"`
	Required  bool   `json:"required"`
}

// FuncObjConfiguration holds the detailed configuration definition of a function node.
type FuncObjConfiguration struct {
	Name     string               `json:"name"`
	FuncDesc string               `json:"funcDesc"`
	Business []DynamicConfigField `json:"business"`
	Inputs   []IOObject           `json:"inputs"`
	Outputs  []IOObject           `json:"outputs"`
	Errors   []*Fault             `json:"errors"`

	// FuncReads declares the static data requirements of the function.
	FuncReads []ContractDef `json:"funcReads,omitempty"`
	// FuncWrites declares the static data production of the function.
	FuncWrites []ContractDef `json:"funcWrites,omitempty"`
}

// FuncObject represents the metadata and configuration definition of a function.
type FuncObject struct {
	ID            string               `json:"id"`   // Unique ID, e.g., "log"
	Name          string               `json:"name"` // Human-readable name, e.g., "Log Message"
	Desc          string               `json:"desc"`
	Dimension     string               `json:"dimension"`
	Tags          []string             `json:"tags"`
	Version       string               `json:"version"`
	Configuration FuncObjConfiguration `json:"configuration"`
}

// NodeFuncObject is the registration unit for a function node.
type NodeFuncObject struct {
	Func       NodeFunc
	FuncObject FuncObject
}

// NodeFuncManager is the interface for managing function nodes.
type NodeFuncManager interface {
	// Register adds a new function node definition.
	Register(f *NodeFuncObject)
	// Get retrieves a function node definition by its ID.
	Get(id string) (*NodeFuncObject, bool)
	// List returns all registered function node objects.
	List() []*NodeFuncObject
}
