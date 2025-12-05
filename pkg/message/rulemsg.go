package message

import (
	"encoding/json"
	"fmt"

	"github.com/NeohetJ/Matrix/internal/contract"
	"github.com/NeohetJ/Matrix/pkg/asset"
	"github.com/NeohetJ/Matrix/pkg/cnst"
	"github.com/NeohetJ/Matrix/pkg/types"
)

// ExtractFromMsg extracts a value from a RuleMsg using the Asset system.
// It supports all schemes registered in the asset package (including rulemsg://).
// This is the recommended replacement for ExtractFromMsgByPath.
func ExtractFromMsg[T any](msg types.RuleMsg, path string) (T, error) {
	var zero T
	if msg == nil {
		return zero, fmt.Errorf("rule message is nil")
	}

	// Use Asset for resolution
	a := asset.Asset[T]{URI: path}
	ctx := asset.NewAssetContext(asset.WithRuleMsg(msg))
	return a.Resolve(ctx)
}

// SetInMsg sets a value in a RuleMsg using the Asset system.
// It supports all schemes registered in the asset package that implement the Set method.
// This is the recommended replacement for SetInMsgByPath.
func SetInMsg[T any](msg types.RuleMsg, path string, value T) error {
	if msg == nil {
		return fmt.Errorf("rule message is nil")
	}

	a := asset.Asset[T]{URI: path}
	ctx := asset.NewAssetContext(asset.WithRuleMsg(msg))
	return a.Set(ctx, value)
}

// ReplaceRuleMsg replaces placeholders in a string using values from RuleMsg.
// It uses asset.RenderTemplate internally.
func ReplaceRuleMsg(template string, msg types.RuleMsg) (string, error) {
	if msg == nil {
		return template, nil
	}
	ctx := asset.NewAssetContext(asset.WithRuleMsg(msg))
	return asset.RenderTemplate(template, ctx)
}

// MsgToMap converts a RuleMsg to a map[string]any.
// It includes Metadata, Data (parsed if JSON), and DataT.
func MsgToMap(msg types.RuleMsg) map[string]any {
	if msg == nil {
		return nil
	}
	res := make(map[string]any)
	res["id"] = msg.ID()
	res["ts"] = msg.Ts()
	res["msgType"] = msg.Type()
	res["dataFormat"] = msg.DataFormat()
	res["metadata"] = msg.Metadata()

	// Data
	if msg.DataFormat() == cnst.JSON {
		var dataMap any
		if err := json.Unmarshal([]byte(msg.Data()), &dataMap); err == nil {
			res["data"] = dataMap
		} else {
			res["data"] = string(msg.Data())
		}
	} else {
		res["data"] = string(msg.Data())
	}

	// DataT
	if dataT := msg.DataT(); dataT != nil {
		dtMap := make(map[string]any)
		for id, item := range dataT.GetAll() {
			dtMap[id] = item.Body()
		}
		res["dataT"] = dtMap
	}

	return res
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
