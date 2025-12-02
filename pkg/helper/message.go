package helper

import (
	"encoding/json"
	"fmt"
	"strings"

	"gitlab.com/neohet/matrix/pkg/types"
	"gitlab.com/neohet/matrix/pkg/utils"
)

// ExtractFromMsgByPath extracts a value from a RuleMsg using a dot-separated path.
// It intelligently handles matrix-specific types like dataT, metadata, and coreObj.
// Example paths: "dataT.my_obj.fieldName", "metadata.key", "'a string literal'".
func ExtractFromMsgByPath(msg types.RuleMsg, path string) (any, bool, error) {
	if msg == nil {
		return nil, false, nil
	}
	if path == "" {
		return msg, true, nil
	}

	// Check for string literals, wrapped in single or double quotes.
	if (strings.HasPrefix(path, "'") && strings.HasSuffix(path, "'")) ||
		(strings.HasPrefix(path, `"`) && strings.HasSuffix(path, `"`)) {
		return path[1 : len(path)-1], true, nil
	}

	parts := strings.Split(path, ".")
	var current any = msg

	for i, part := range parts {
		// 1. Handle top-level keywords on the RuleMsg itself
		if i == 0 {
			if ruleMsg, ok := current.(types.RuleMsg); ok {
				switch part {
				case "dataT":
					current = ruleMsg.DataT()
					continue
				case "metadata":
					current = ruleMsg.Metadata()
					continue
				case "data":
					current = ruleMsg.Data()
					continue
				}
			}
		}

		// 2. Handle special matrix types
		if dataT, ok := current.(types.DataT); ok {
			coreObj, found := dataT.Get(part)
			if !found {
				return nil, false, nil
			}
			// If it's the last part of the path, return the whole CoreObj body.
			if i == len(parts)-1 {
				current = coreObj.Body()
			} else {
				// Otherwise, continue traversal into the object's body.
				current = coreObj.Body()
			}
			continue
		}
		if metadata, ok := current.(types.Metadata); ok {
			val, exists := metadata[part]
			if !exists {
				return nil, false, nil
			}
			// Metadata values are final, no further path traversal
			if i < len(parts)-1 {
				return nil, false, fmt.Errorf("cannot extract path beyond metadata key '%s'", part)
			}
			return val, true, nil
		}

		// 3. If not a special type, fall back to generic reflection-based extraction
		var found bool
		var err error
		current, found, err = utils.ExtractByPath(current, part)
		if err != nil {
			return nil, false, fmt.Errorf("error during generic extraction at part '%s': %w", part, err)
		}
		if !found {
			return nil, false, nil
		}
	}

	return current, true, nil
}

// BuildDataSource creates a nested map from a RuleMsg for placeholder substitution.
// The structure is designed to be used with a path-aware placeholder replacement function.
// - data: from msg.Data()
// - meta: from msg.Metadata()
// - dataT: from msg.DataT()
func BuildDataSource(msg types.RuleMsg) map[string]any {
	dataSource := make(map[string]any)

	// 1. Populate from msg.Data (if it's a JSON object)
	var dataMap map[string]any
	if msg.Data() != "" {
		if err := json.Unmarshal([]byte(msg.Data()), &dataMap); err == nil {
			dataSource["data"] = dataMap
		} else {
			// If not a valid JSON, treat it as a raw string.
			dataSource["data"] = msg.Data()
		}
	}

	// 2. Populate from msg.Metadata
	if meta := msg.Metadata(); meta != nil {
		dataSource["metadata"] = meta
	}

	// 3. Populate from msg.DataT, creating a nested map of objects.
	if dataT := msg.DataT(); dataT != nil {
		dataTMap := make(map[string]any)
		if allItems, ok := dataT.(interface {
			GetAll() map[string]types.CoreObj
		}); ok {
			for objId, coreObj := range allItems.GetAll() {
				if body := coreObj.Body(); body != nil {
					dataTMap[objId] = body
				}
			}
		}
		if len(dataTMap) > 0 {
			dataSource["dataT"] = dataTMap
		}
	}

	return dataSource
}
