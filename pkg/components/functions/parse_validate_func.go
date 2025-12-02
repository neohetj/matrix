package functions

import (
	"encoding/json"
	"fmt"

	"gitlab.com/neohet/matrix/pkg/helper"
	"gitlab.com/neohet/matrix/pkg/registry"
	"gitlab.com/neohet/matrix/pkg/types"
)

const (
	ParseValidateFuncID = "parseValidate"
)

func init() {
	registry.Default.NodeFuncManager.Register(&types.NodeFuncObject{
		Func: ParseValidateFunc,
		FuncObject: types.FuncObject{
			ID:        ParseValidateFuncID,
			Name:      "Parse And Validate",
			Desc:      "Parses a JSON string from msg.Data into a specified CoreObj and stores it in msg.DataT.",
			Dimension: "Transformation",
			Tags:      []string{"parse", "json", "validate"},
			Version:   "1.0.0",
			Configuration: types.FuncObjConfiguration{
				Name:     "Parse and Validate Config",
				FuncDesc: "Configure the target CoreObj for parsing.",
				Business: []types.DynamicConfigField{
					{ID: "targetCoreObjSid", Name: "Target CoreObj SID", Desc: "The System ID of the CoreObj to parse into (e.g., sid://my_app/types.MyData).", Required: true, Type: "string"},
					{ID: "targetObjId", Name: "Target CoreObj Key", Desc: "The key used to store the parsed object in msg.DataT.", Required: true, Type: "string"},
					// TODO: Add validation rules support in the future.
				},
			},
		},
	})
}

// ParseValidateFunc is a NodeFunc that parses a JSON string from msg.Data into a CoreObj.
func ParseValidateFunc(ctx types.NodeCtx, msg types.RuleMsg) {
	bizConfig, ok := helper.GetBusinessConfig(ctx)
	if !ok {
		ctx.HandleError(msg, types.ErrInvalidParams.Wrap(fmt.Errorf("business config not found")))
		return
	}

	sid, _ := bizConfig["targetCoreObjSid"].(string)
	key, _ := bizConfig["targetObjId"].(string)

	if sid == "" || key == "" {
		ctx.HandleError(msg, types.ErrInvalidParams.Wrap(fmt.Errorf("targetCoreObjSid and targetObjId are required")))
		return
	}

	// Create a new CoreObj instance within the message's DataT container.
	// This is the standard way to create and register a new data object in the message.
	coreObjInstance, err := msg.DataT().NewItem(sid, key)
	if err != nil {
		ctx.HandleError(msg, types.ErrInvalidParams.Wrap(fmt.Errorf("failed to create new CoreObj with SID %s and key %s: %w", sid, key, err)))
		return
	}

	// Unmarshal the JSON string from msg.Data into the newly created CoreObj's body.
	if err := json.Unmarshal([]byte(msg.Data()), coreObjInstance.Body()); err != nil {
		ctx.HandleError(msg, types.ErrInvalidParams.Wrap(fmt.Errorf("failed to unmarshal JSON to CoreObj body: %w", err)))
		return
	}

	// TODO: Implement validation logic here based on validation rules in config.
	// For example:
	// if validator, ok := coreObjInstance.(types.Validator); ok {
	//     if err := validator.Validate(); err != nil {
	//         ctx.HandleError(msg, types.ErrInvalidData.Wrap(err))
	//         return
	//     }
	// }

	ctx.TellSuccess(msg)
}
