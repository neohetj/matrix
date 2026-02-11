package helper

import (
	"fmt"
	"mime/multipart"

	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/message"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/pkg/utils"
)

// ProcessInbound handles mapping from an external source (like HTTP request parts) TO RuleMsg.
// Used by: HttpEndpoint (Request), HttpClient (Response).
func ProcessInbound(ctx types.NodeCtx, msg types.RuleMsg, packet types.EndpointIOPacket, provider ValueProvider) error {
	// 1. MapAll: Map the entire packet to a single path in RuleMsg
	if packet.MapAll != nil && *packet.MapAll != "" {
		val, ok, err := provider.GetAll()
		if err != nil {
			return MapAllDataFailed.Wrap(fmt.Errorf("from provider: %w", err))
		}
		if ok {
			if err := message.SetInMsg(msg, *packet.MapAll, val); err != nil {
				return MapAllDataFailed.Wrap(fmt.Errorf("to %s: %w", *packet.MapAll, err))
			}
		}
	}

	// 2. Fields: Map individual fields
	for _, field := range packet.Fields {
		rawVal, found, err := provider.GetValue(field.Name)
		if err != nil {
			return FieldConversionFailed.Wrap(fmt.Errorf("from provider field '%s': %w", field.Name, err))
		}
		if !found {
			if field.Required && field.DefaultValue == nil {
				return RequiredFieldMissing.Wrap(fmt.Errorf("'%s'", field.Name))
			} else {
				found = true
				rawVal = field.DefaultValue
			}
		}

		if found {
			convertedVal, err := convertValue(rawVal, field.Type)
			if err != nil {
				return FieldConversionFailed.Wrap(fmt.Errorf("field '%s': %w", field.Name, err))
			}
			if err := message.SetInMsg(msg, field.BindPath, convertedVal); err != nil {
				return SetFieldFailed.Wrap(fmt.Errorf("field '%s' to %s: %w", field.Name, field.BindPath, err))
			}
		}
	}

	return nil
}

// ProcessOutbound handles mapping FROM RuleMsg TO an external target (like HTTP request parts).
// Used by: HttpEndpoint (Response), HttpClient (Request).
// It returns a map of values or a single value (e.g. slice) that should be sent to the external target.
func ProcessOutbound(ctx types.NodeCtx, msg types.RuleMsg, packet types.EndpointIOPacket, provider ValueProvider) (any, error) {
	result := make(map[string]any)
	var rootValue any

	// 1. MapAll: Extract a whole object from RuleMsg
	if packet.MapAll != nil && *packet.MapAll != "" {
		val, found, err := provider.GetValue(*packet.MapAll)
		if err != nil {
			return nil, ExtractMapAllFailed.Wrap(fmt.Errorf("from provider: %w", err))
		}
		if found {
			// If the value is a map, merge it.
			valMap, err := utils.ToMap(val)
			if err == nil {
				for k, v := range valMap {
					result[k] = v
				}
			} else {
				// If it's NOT a map (e.g. Slice, primitive), treat as root value.
				// But strict check: if we have fields defined, we cannot have a non-map root value.
				if len(packet.Fields) > 0 {
					return nil, ExtractMapAllFailed.Wrap(fmt.Errorf("MapAll points to non-object type %T, but Fields are defined", val))
				}
				rootValue = val
			}
		}
	}

	if rootValue != nil {
		return rootValue, nil
	}

	// 2. Fields: Extract individual fields and override/add to result
	for _, field := range packet.Fields {
		var val any
		var found bool
		var err error

		if field.BindPath == "" {
			val = field.DefaultValue
			found = true
		} else {
			val, found, err = provider.GetValue(field.BindPath)
			if err != nil {
				ctx.Warn("Failed to extract field", "bindPath", field.BindPath, "error", err)
				continue
			}
		}

		if !found {
			if field.Required && field.DefaultValue == nil {
				return nil, RequiredFieldMissing.Wrap(fmt.Errorf("'%s' (bound to %s)", field.Name, field.BindPath))
			}
			if field.DefaultValue != nil {
				val = field.DefaultValue
				found = true
				ctx.Debug("Using DefaultValue", "field", field.Name, "value", val)
			}
		}

		if found {
			ctx.Debug("Field found", "field", field.Name, "value", val)
			convertedVal, err := convertValue(val, field.Type)
			if err != nil {
				return nil, FieldConversionFailed.Wrap(fmt.Errorf("field '%s': %w", field.Name, err))
			}
			utils.SetValueByDotPath(result, field.Name, convertedVal)
		}
	}

	return result, nil
}

func convertValue(val any, targetType cnst.MType) (any, error) {
	if targetType == "" {
		return val, nil
	}
	// Special handling for FILE type
	if targetType == cnst.FILE {
		if _, ok := val.(*multipart.FileHeader); ok {
			return val, nil
		}
		return nil, ExpectedFileType.Wrap(fmt.Errorf("got %T", val))
	}

	// Dereference pointer if needed
	if val != nil {
		if ptr, ok := val.(*bool); ok {
			if ptr != nil {
				val = *ptr
			}
		} else if ptr, ok := val.(*string); ok {
			if ptr != nil {
				val = *ptr
			}
		} else if ptr, ok := val.(*int); ok {
			if ptr != nil {
				val = *ptr
			}
		} else if ptr, ok := val.(*int64); ok {
			if ptr != nil {
				val = *ptr
			}
		} else if ptr, ok := val.(*float64); ok {
			if ptr != nil {
				val = *ptr
			}
		}
	}

	return utils.Convert(val, targetType)
}
