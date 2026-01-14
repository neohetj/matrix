package message

import (
	"encoding/json"
	"fmt"

	"github.com/neohetj/matrix/pkg/asset"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
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
