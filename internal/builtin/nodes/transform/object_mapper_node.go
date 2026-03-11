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

package transform

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/asset"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/helper"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/pkg/utils"
)

func init() {
	registry.Default.NodeManager.Register(ObjectMapperNodePrototype)
	registry.Default.FaultRegistry.Register(ObjectMapperNodePrototype.Errors()...)
}

const (
	ObjectMapperNodeType = "transform/object_mapper"
)

// ObjectMapperNodeConfiguration defines the configuration for the ObjectMapper node.
// It maps source data (from RuleMsg) to a target CoreObj.
type ObjectMapperNodeConfiguration struct {
	// MappingDefinition defines how to map fields from source to target.
	MappingDefinition types.EndpointIOPacket `json:"mappingDefinition"`
}

type ObjectMapperNode struct {
	types.BaseNode
	types.Instance
	nodeConfig ObjectMapperNodeConfiguration
}

// ObjectMapperNodePrototype is the shared prototype instance used for registration.
// Exported for centralized registration in builtin/init.go.
var ObjectMapperNodePrototype = &ObjectMapperNode{
	BaseNode: *types.NewBaseNode(ObjectMapperNodeType, types.NodeMetadata{
		Name:        "Object Mapper",
		Description: "Maps data from source (JSON/CoreObj) to a target CoreObj using defined rules.",
		Dimension:   "Transformation",
		Tags:        []string{"map", "transform", "coreobj"},
		Version:     "1.0.0",
	}),
}

func (n *ObjectMapperNode) New() types.Node {
	return &ObjectMapperNode{
		BaseNode: n.BaseNode,
	}
}

func (n *ObjectMapperNode) Init(cfg types.ConfigMap) error {
	if err := utils.Decode(cfg, &n.nodeConfig); err != nil {
		return fmt.Errorf("failed to decode object mapper node config: %w", err)
	}
	return nil
}

// MessageValueProvider implements helper.ValueProvider for RuleMsg mapping.
type MessageValueProvider struct {
	ctx types.NodeCtx
	msg types.RuleMsg
}

// cleanAndParseString 尝试清理字符串中的 Markdown 标记并解析为 JSON 对象。
// 如果解析失败，则返回原始字符串（如果是 JSON 对象/数组格式但解析失败，则返回清理后的字符串）。
func cleanAndParseString(val string) any {
	strVal := val
	// 1. 处理Markdown代码块标记
	if strings.Contains(strVal, "```json") {
		strVal = strings.TrimPrefix(strVal, "```json")
		strVal = strings.TrimSuffix(strVal, "```")
		strVal = strings.TrimSpace(strVal)
	} else if strings.Contains(strVal, "```") {
		// 处理可能的非 json 标记代码块
		strVal = strings.Trim(strVal, "`")
		strVal = strings.TrimSpace(strVal)
	}

	// 2. 尝试解析为对象
	// 只有当看起来像 JSON 对象或数组时才尝试解析，避免普通字符串被误判
	trimmed := strings.TrimSpace(strVal)
	if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
		(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
		var parsedVal any
		if err := json.Unmarshal([]byte(trimmed), &parsedVal); err == nil {
			return parsedVal
		}
	}

	// 如果不是 JSON 或解析失败，但经过了清理（例如去除了 markdown 标记），
	// 我们返回清理后的字符串，以便后续处理可能仍然需要它
	return strVal
}

func (p *MessageValueProvider) GetValue(path string) (any, bool, error) {
	if !asset.IsURI(path) {
		return nil, false, nil
	}

	val, found, err := helper.RuleMsgProvider{Msg: p.msg}.GetValue(path)
	if err != nil {
		p.ctx.Warn("failed to extract value from message", "path", path, "error", err)
		return nil, false, err
	}
	if !found {
		return nil, false, nil
	}

	// 如果值是字符串，尝试解析为JSON对象
	var targetStr string
	isString := false

	switch v := val.(type) {
	case string:
		targetStr = v
		isString = true
	case *string:
		if v != nil {
			targetStr = *v
			isString = true
		}
	}

	if isString {
		return cleanAndParseString(targetStr), true, nil
	}

	return val, true, nil
}

func (p *MessageValueProvider) GetAll() (any, bool, error) {
	// MapAll 通常映射整个 msg.Data()
	data := p.msg.Data()
	if data == "" {
		return nil, false, nil
	}

	// 使用通用的清理和解析逻辑
	return cleanAndParseString(string(data)), true, nil
}

func (n *ObjectMapperNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {
	provider := &MessageValueProvider{ctx: ctx, msg: msg}

	if err := helper.ProcessInbound(ctx, msg, n.nodeConfig.MappingDefinition, provider); err != nil {
		ctx.HandleError(msg, types.InternalError.Wrap(fmt.Errorf("mapping failed: %w", err)))
		return
	}

	ctx.TellSuccess(msg)
}

func (n *ObjectMapperNode) Destroy() {}

func (n *ObjectMapperNode) DataContract() types.DataContract {
	contract := types.DataContract{
		Reads:  make([]string, 0),
		Writes: make([]string, 0),
	}

	appendURI := func(uri string, isRead bool) {
		normalized := asset.NormalizeURI(uri)
		if normalized == "" {
			return
		}
		if isRead {
			contract.Reads = append(contract.Reads, normalized)
		} else {
			contract.Writes = append(contract.Writes, normalized)
		}
	}

	for _, field := range n.nodeConfig.MappingDefinition.Fields {
		if field.Name != "" {
			appendURI(field.Name, true)
		}
		if field.BindPath != "" {
			appendURI(field.BindPath, false)
		}
	}

	if n.nodeConfig.MappingDefinition.MapAll != nil && *n.nodeConfig.MappingDefinition.MapAll != "" {
		appendURI(*n.nodeConfig.MappingDefinition.MapAll, false)
		// Default read is full data if mapping all
		contract.Reads = append(contract.Reads, asset.DataURI(cnst.JSON))
	}

	return contract
}
