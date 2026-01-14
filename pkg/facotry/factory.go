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

package facotry

import (
	"context"
	"fmt"

	"github.com/neohetj/matrix/internal/runtime"

	"github.com/neohetj/matrix/internal/contract"
	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/types"
)

// NewDataT creates a new instance of the default DataT implementation.
func NewDataT() types.DataT {
	// TODO: 支持使用第三方定义的DataT
	return contract.NewDataT()
}

// NewCoreObj creates a new instance of the default CoreObj implementation.
func NewCoreObj(key string, def types.CoreObjDef) types.CoreObj {
	return contract.NewDefaultCoreObj(key, def)
}

// NewCoreObj creates a new instance of the default CoreObj implementation.
func NewCoreObjDef(prototype any, sid, desc string) types.CoreObjDef {
	return contract.NewDefaultCoreObjDef(prototype, sid, desc)
}

// NewMsg creates a new message with the given type, data, and metadata.
// It automatically generates a new UUID and sets the timestamp.
// The dataFormat is initially empty and should be set explicitly via WithDataFormat.
func NewMsg(msgType, data string, metadata types.Metadata, dataT types.DataT) types.RuleMsg {
	if dataT == nil {
		dataT = NewDataT()
	}
	return contract.NewDefaultRuleMsg(msgType, data, metadata, dataT)
}

func NewMinNodeCtx(nodeID string) types.NodeCtx {
	return registry.NewMinimalNodeCtx(nodeID)
}

// NewNodeCtx creates a new node context.
func NewNodeCtx(ctx context.Context, r types.Runtime, chain types.ChainInstance, selfDef *types.NodeDef, parent types.NodeCtx, onEnd func(msg types.RuleMsg, err error), aspects []types.Aspect, callback types.CallbackFunc) types.NodeCtx {
	// Type assertion is required because NewDefaultNodeCtx expects a concrete type, not an interface.
	defaultRuntime, _ := r.(*runtime.DefaultRuntime)
	defaultParent, _ := parent.(*runtime.DefaultNodeCtx)
	return runtime.NewDefaultNodeCtx(ctx, defaultRuntime, chain, selfDef, defaultParent, onEnd, aspects, callback)
}

// NewSubMsg creates a new sub-message from a parent message.
// The new message type is constructed by combining the parent's ID with the sub-chain ID.
// This allows for tracking the hierarchy of messages in trace logs.
func NewSubMsg(parentMsg types.RuleMsg, subChainId string) types.RuleMsg {
	if parentMsg == nil {
		return NewMsg(subChainId, "", nil, nil)
	}

	newType := fmt.Sprintf("%s::%s", parentMsg.Type(), subChainId)
	if parentMsg.Type() == "" {
		// If parent type is empty, use parent ID as base
		newType = fmt.Sprintf("%s::%s", parentMsg.ID(), subChainId)
	}

	// If parent type already contains "::", it means it's already a sub-message.
	// We append to it to maintain full path.

	// Create new message with derived type
	return NewMsg(newType, "", parentMsg.Metadata().Copy(), NewDataT())
}
