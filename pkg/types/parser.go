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

// Package parser defines the interface for parsing the rule chain DSL.
// Author: Neohet
package types

import "encoding/json"

// Connection defines the link between two nodes in the rule chain.
type Connection struct {
	FromID string `json:"fromId"`
	ToID   string `json:"toId"`
	Type   string `json:"type"`
}

// Relation defines a logical link between two nodes for visualization and modeling.
type Relation struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label"`
}

// TODO：将这些定义都迁移到types中
// ViewType is a string enum for different rule chain visualization types.
type ViewType string

const (
	// ViewTypeStaticTopology represents a static, non-executable graph showing deployment or logical relationships.
	// It primarily uses the 'relations' field for visualization.
	ViewTypeStaticTopology ViewType = "static-topology"

	// ViewTypeExecutionFlow represents an executable DAG (Directed Acyclic Graph) that defines a workflow.
	// It primarily uses the 'connections' field for visualization.
	ViewTypeExecutionFlow ViewType = "execution-flow"

	// ViewTypeHybrid represents a view that combines both logical relations and execution connections.
	ViewTypeHybrid ViewType = "hybrid"
)

// RuleChainAttrs holds attributes about the rule chain definition.
type RuleChainAttrs struct {
	Executable bool     `json:"executable"`
	ViewType   ViewType `json:"viewType,omitempty"`
	Imports    []string `json:"imports,omitempty"`
}

// RuleChainData holds the core data of a rule chain.
type RuleChainData struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	Configuration map[string]any `json:"configuration,omitempty"`
	Attrs         RuleChainAttrs `json:"attrs,omitempty"`
}

// UnmarshalJSON implements the json.Unmarshaler interface to set default values for RuleChainData.
func (r *RuleChainData) UnmarshalJSON(data []byte) error {
	// To avoid recursion, we use a temporary type that does not have the UnmarshalJSON method.
	type Alias RuleChainData

	// Set default values before unmarshaling.
	temp := &Alias{
		Attrs: RuleChainAttrs{
			Executable: true, // Default value for Executable
		},
	}

	if err := json.Unmarshal(data, temp); err != nil {
		return err
	}

	*r = RuleChainData(*temp)
	return nil
}

// MetadataData holds the metadata of a rule chain, such as nodes and connections.
type MetadataData struct {
	Nodes       []NodeDef    `json:"nodes"`
	Connections []Connection `json:"connections"`
	Relations   []Relation   `json:"relations,omitempty"`
}

// RuleChainDef represents the definition of a rule chain.
type RuleChainDef struct {
	RuleChain RuleChainData `json:"ruleChain"`
	Metadata  MetadataData  `json:"metadata"`
}

// NodeDef represents the definition of a single node in the rule chain.
type NodeDef struct {
	ID            string         `json:"id"`
	Type          string         `json:"type"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	Configuration map[string]any `json:"configuration"`
	Inputs        map[string]any `json:"inputs,omitempty"`
	Outputs       map[string]any `json:"outputs,omitempty"`
}

// Parser is the interface for parsing the rule chain definition file (DSL).
// It allows for different DSL formats (e.g., JSON, YAML) to be used.
type Parser interface {
	// DecodeRuleChain parses a rule chain structure from a byte slice.
	DecodeRuleChain(dsl []byte) (*RuleChainDef, error)

	// DecodeNode parses a single node structure from a byte slice.
	DecodeNode(dsl []byte) (*NodeDef, error)

	// EncodeRuleChain converts a rule chain structure into a byte slice.
	EncodeRuleChain(def *RuleChainDef) ([]byte, error)

	// EncodeNode converts a single node structure into a byte slice.
	EncodeNode(def *NodeDef) ([]byte, error)
}
