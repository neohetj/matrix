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

package types

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
	Errors   []*ServiceError      `json:"errors"`

	// ReadsData (可选) 声明函数从原始 RuleMsg.Data 中读取的字段路径列表。
	ReadsData []string `json:"readsData,omitempty"`
	// ReadsMetadata (可选) 声明函数读取的元数据键。
	ReadsMetadata []MetadataDef `json:"readsMetadata,omitempty"`
	// WritesMetadata (可选) 声明函数写入的元数据键。
	WritesMetadata []MetadataDef `json:"writesMetadata,omitempty"`
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
