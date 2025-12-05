package helper

import (
	"fmt"
	"mime/multipart"

	"github.com/NeohetJ/Matrix/pkg/cnst"
	"github.com/NeohetJ/Matrix/pkg/message"
	"github.com/NeohetJ/Matrix/pkg/types"
	"github.com/NeohetJ/Matrix/pkg/utils"
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
// It returns a map of values that should be sent to the external target.
func ProcessOutbound(ctx types.NodeCtx, msg types.RuleMsg, packet types.EndpointIOPacket, provider ValueProvider) (map[string]any, error) {
	result := make(map[string]any)

	// 1. MapAll: Extract a whole object from RuleMsg and merge it into result
	if packet.MapAll != nil && *packet.MapAll != "" {
		val, found, err := provider.GetValue(*packet.MapAll)
		if err != nil {
			return nil, ExtractMapAllFailed.Wrap(fmt.Errorf("from provider: %w", err))
		}
		if found {
			// If the value is a map, merge it. If it's something else (like []byte or string body),
			// we might handle it differently depending on context.
			// For generic "MapAll", we assume it provides a base set of fields.
			valMap, err := utils.ToMap(val)
			if err == nil {
				for k, v := range valMap {
					result[k] = v
				}
			} else {
				// If not a map, perhaps it's a raw body?
				// For now, we put it under a special empty key or let the caller handle it?
				// Strategy: If MapAll points to a non-map, it might be the WHOLE body content.
				// We return it with a specific internal key "" to indicate "this is the raw value".
				result[""] = val
			}
		}
	}

	// 2. Fields: Extract individual fields and override/add to result
	for _, field := range packet.Fields {
		val, found, err := provider.GetValue(field.BindPath)
		if err != nil {
			ctx.Warn("Failed to extract field", "bindPath", field.BindPath, "error", err)
			continue
		}
		if !found {
			if field.Required {
				return nil, RequiredFieldMissing.Wrap(fmt.Errorf("'%s' (bound to %s)", field.Name, field.BindPath))
			}
			if field.DefaultValue != nil {
				val = field.DefaultValue
				found = true
			}
		}

		if found {
			convertedVal, err := convertValue(val, field.Type)
			if err != nil {
				return nil, FieldConversionFailed.Wrap(fmt.Errorf("field '%s': %w", field.Name, err))
			}
			// Support dot notation in field.Name for nested structure construction?
			// For simplicity, we use SetValueByDotPath if the caller supports it, but here we return a flat map
			// where keys might contain dots. The consumer (e.g., JSON marshaller) should handle structure.
			// Ideally, utils.SetValueByDotPath should be used on the result map.
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

	return utils.Convert(val, targetType)
}
