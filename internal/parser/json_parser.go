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

package parser

import (
	"encoding/json"

	"github.com/NeohetJ/Matrix/pkg/types"
)

// JsonParser is the default implementation of the Parser interface, using standard JSON.
type JsonParser struct{}

// NewJsonParser creates a new instance of JsonParser.
func NewJsonParser() *JsonParser {
	return &JsonParser{}
}

// DecodeRuleChain parses a rule chain structure from a JSON byte slice.
func (p *JsonParser) DecodeRuleChain(dsl []byte) (*types.RuleChainDef, error) {
	var def types.RuleChainDef
	if err := json.Unmarshal(dsl, &def); err != nil {
		return nil, err
	}
	return &def, nil
}

// DecodeNode parses a single node structure from a JSON byte slice.
func (p *JsonParser) DecodeNode(dsl []byte) (*types.NodeDef, error) {
	var def types.NodeDef
	if err := json.Unmarshal(dsl, &def); err != nil {
		return nil, err
	}
	return &def, nil
}

// EncodeRuleChain converts a rule chain structure into a JSON byte slice.
func (p *JsonParser) EncodeRuleChain(def *types.RuleChainDef) ([]byte, error) {
	return json.Marshal(def)
}

// EncodeNode converts a single node structure into a JSON byte slice.
func (p *JsonParser) EncodeNode(def *types.NodeDef) ([]byte, error) {
	return json.Marshal(def)
}
