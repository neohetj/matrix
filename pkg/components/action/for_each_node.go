package action

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"gitlab.com/neohet/matrix/pkg/helper"
	"gitlab.com/neohet/matrix/pkg/registry"
	"gitlab.com/neohet/matrix/pkg/types"
	"gitlab.com/neohet/matrix/pkg/utils"
)

const (
	ForEachNodeType       = "action/forEach"
	MetadataKeyLoopIndex  = "_loopIndex"
	MetadataKeyIsLastItem = "is_last_item"
)

var forEachNodePrototype = &ForEachNode{
	BaseNode: *types.NewBaseNode(ForEachNodeType, types.NodeDefinition{
		Name:        "For Each",
		Description: "Iterates over a list from msg.DataT or an expression, and executes a sub-chain for each item.",
		Dimension:   "Action",
		Tags:        []string{"action", "loop", "iterator"},
		Version:     "1.0.0",
		WritesMetadata: []types.MetadataDef{
			{Key: MetadataKeyLoopIndex, Description: "The 0-based index of the current loop iteration. This is added to the message sent to the sub-chain."},
			{Key: MetadataKeyIsLastItem, Description: "A boolean (as a string 'true' or 'false') indicating if it's the last item in the loop. This is added to the message sent to the sub-chain."},
		},
	}),
}

func init() {
	registry.Default.NodeManager.Register(forEachNodePrototype)
}

// InputMappingConfig defines the configuration for a single input mapping.
type InputMappingConfig struct {
	// From specifies the source of the data in the parent message.
	// Examples: "dataT.objId.field", "metadata.key", "data.field", "_item" (for current loop item).
	From string `json:"from"`
	// To specifies the destination of the data in the sub-chain's message.
	// Examples: "dataT.targetParam", "metadata.targetKey", "data".
	To string `json:"to"`
	// DefineSID is required if 'To' maps to a dataT object (e.g., "dataT.targetParam").
	DefineSID string `json:"defineSid,omitempty"`
}

// ForEachNodeConfiguration holds the instance-specific configuration.
type ForEachNodeConfiguration struct {
	// ItemsExpression is an expression to get the list to iterate over. e.g., "dataT.myList.items"
	ItemsExpression string `json:"itemsExpression"`
	// ChainId is the ID of the sub-chain to execute for each item.
	ChainId string `json:"chainId"`
	// Async specifies whether to execute the sub-chains asynchronously.
	Async bool `json:"async,omitempty"`
	// ContinueOnError specifies whether to continue the loop even if a sub-chain execution fails.
	// This only applies to synchronous execution.
	ContinueOnError bool `json:"continueOnError,omitempty"`
	// MessageScope defines the lifecycle of the RuleMsg across iterations.
	// Supported values: "INDEPENDENT" (default), "SHARED".
	MessageScope string `json:"messageScope,omitempty"`
	// InputMappings defines how to map data from the parent message to the sub-chain's message.
	InputMappings []InputMappingConfig `json:"inputMappings"`
}

// ForEachNode is a component that iterates over a list and executes a sub-chain.
type ForEachNode struct {
	types.BaseNode
	types.Instance
	nodeConfig ForEachNodeConfiguration
}

// New creates a new instance of ForEachNode.
func (n *ForEachNode) New() types.Node {
	return &ForEachNode{
		BaseNode: n.BaseNode,
	}
}

// Type returns the node type.
func (n *ForEachNode) Type() types.NodeType {
	return ForEachNodeType
}

// Init initializes the node instance.
func (n *ForEachNode) Init(configuration types.Config) error {
	if err := utils.Decode(configuration, &n.nodeConfig); err != nil {
		return fmt.Errorf("failed to decode forEach node config: %w", err)
	}
	if n.nodeConfig.ChainId == "" {
		return fmt.Errorf("'chainId' is not specified for node %s", n.ID())
	}
	if n.nodeConfig.ItemsExpression == "" {
		return fmt.Errorf("'itemsExpression' is not specified for node %s", n.ID())
	}
	return nil
}

// OnMsg executes the loop.
func (n *ForEachNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {
	// 1. Get the target sub-chain runtime
	targetRuntime, ok := registry.Default.RuntimePool.Get(n.nodeConfig.ChainId)
	if !ok {
		ctx.HandleError(msg, fmt.Errorf("target chain with id '%s' not found", n.nodeConfig.ChainId))
		return
	}

	// 2. Extract the list of items to iterate over
	itemsRaw, found, err := helper.ExtractFromMsgByPath(msg, n.nodeConfig.ItemsExpression)
	if err != nil {
		ctx.HandleError(msg, fmt.Errorf("failed to extract items with expression '%s': %w", n.nodeConfig.ItemsExpression, err))
		return
	}
	if !found {
		ctx.HandleError(msg, fmt.Errorf("items not found with expression '%s'", n.nodeConfig.ItemsExpression))
		return
	}

	// Handle the case where the expression returns a JSON array as a raw byte slice.
	if rawMessage, ok := itemsRaw.(json.RawMessage); ok {
		var sliceOfItems []interface{}
		if err := json.Unmarshal(rawMessage, &sliceOfItems); err != nil {
			ctx.HandleError(msg, fmt.Errorf("failed to unmarshal itemsExpression result from json.RawMessage: %w", err))
			return
		}
		itemsRaw = sliceOfItems // Replace the raw message with the actual slice.
	}

	itemsVal := reflect.ValueOf(itemsRaw)
	if itemsVal.Kind() == reflect.Ptr {
		itemsVal = itemsVal.Elem()
	}
	if itemsVal.Kind() != reflect.Slice {
		ctx.HandleError(msg, fmt.Errorf("expression '%s' did not return a slice, but a %s", n.nodeConfig.ItemsExpression, itemsVal.Kind()))
		return
	}

	// 3. Iterate and execute
	itemCount := itemsVal.Len()
	ctx.Info("Starting forEach loop", "chainId", n.nodeConfig.ChainId, "itemCount", itemCount, "async", n.nodeConfig.Async, "messageScope", n.nodeConfig.MessageScope)

	if n.nodeConfig.Async {
		// Asynchronous execution (always uses INDEPENDENT message scope)
		var wg sync.WaitGroup
		for i := 0; i < itemCount; i++ {
			wg.Add(1)
			go func(index int, item interface{}) {
				defer wg.Done()
				// INDEPENDENT: Create a new message and apply all mappings
				iterMsg := types.NewMsg(msg.ID(), "", msg.Metadata().Copy(), types.NewDataT()).WithDataFormat(types.UNKNOWN)
				for _, mappingConfig := range n.nodeConfig.InputMappings {
					if err := n.applyMapping(ctx, msg, iterMsg, mappingConfig, item); err != nil {
						ctx.Error("Async sub-chain mapping failed, skipping iteration.", "iteration", index, "error", err)
						return
					}
				}
				// Execute the sub-chain
				if _, err := n.executeSubChain(ctx, iterMsg, index, itemCount, targetRuntime); err != nil {
					// In async mode, we log the error but don't stop other iterations.
					ctx.Error("Async sub-chain execution failed", "iteration", index, "error", err)
				}
			}(i, itemsVal.Index(i).Interface())
		}
		wg.Wait()
		// In async mode, the parent chain always succeeds after launching all sub-chains.
		ctx.TellSuccess(msg)
	} else {
		// Synchronous execution
		var sharedIterMsg types.RuleMsg
		if n.nodeConfig.MessageScope == "SHARED" {
			// Create the shared message once before the loop.
			sharedIterMsg = types.NewMsg(msg.ID(), "", msg.Metadata().Copy(), types.NewDataT()).WithDataFormat(types.UNKNOWN)
			// Pre-populate it with non-item mappings
			for _, mappingConfig := range n.nodeConfig.InputMappings {
				if mappingConfig.From != "_item" {
					if err := n.applyMapping(ctx, msg, sharedIterMsg, mappingConfig, nil); err != nil {
						ctx.HandleError(msg, fmt.Errorf("failed to apply initial mapping for shared message: %w", err))
						return
					}
				}
			}
		}

		for i := 0; i < itemCount; i++ {
			var iterMsg types.RuleMsg
			item := itemsVal.Index(i).Interface()

			if n.nodeConfig.MessageScope == "SHARED" {
				iterMsg = sharedIterMsg // Reuse the shared message
				// Find the "_item" mapping and apply it for the current iteration
				for _, mappingConfig := range n.nodeConfig.InputMappings {
					if mappingConfig.From == "_item" {
						if err := n.applyMapping(ctx, msg, iterMsg, mappingConfig, item); err != nil {
							ctx.HandleError(msg, fmt.Errorf("failed to apply item mapping for shared message: %w", err))
							return
						}
						break // Assume only one _item mapping
					}
				}
			} else {
				// INDEPENDENT (default): Create a new message and apply all mappings
				iterMsg = types.NewMsg(msg.ID(), "", msg.Metadata().Copy(), types.NewDataT()).WithDataFormat(types.UNKNOWN)
				for _, mappingConfig := range n.nodeConfig.InputMappings {
					if err := n.applyMapping(ctx, msg, iterMsg, mappingConfig, item); err != nil {
						ctx.HandleError(msg, fmt.Errorf("failed to apply mapping for independent message: %w", err))
						return
					}
				}
			}

			finalIterMsg, err := n.executeSubChain(ctx, iterMsg, i, itemCount, targetRuntime)
			if err != nil {
				ctx.Error("Sync sub-chain execution failed (error object returned).", "iteration", i, "error", err, "continueOnError", n.nodeConfig.ContinueOnError)
				if !n.nodeConfig.ContinueOnError {
					ctx.TellFailure(finalIterMsg, fmt.Errorf("forEach loop failed at iteration %d: %w", i, err))
					return
				}
				// If ContinueOnError is true, just log and continue the loop.
				continue
			}
			// Also check for failures reported in metadata.
			if errMsg, ok := finalIterMsg.Metadata()["error"]; ok {
				ctx.Error("Sync sub-chain execution failed (error in metadata).", "iteration", i, "error", errMsg, "continueOnError", n.nodeConfig.ContinueOnError)
				if !n.nodeConfig.ContinueOnError {
					ctx.TellFailure(finalIterMsg, fmt.Errorf("forEach loop failed at iteration %d: %s", i, errMsg))
					return
				}
				// If ContinueOnError is true, just log and continue the loop.
				continue
			}
		}
		ctx.Info("ForEach loop completed successfully")
		ctx.TellSuccess(msg)
	}
}

// applyMapping populates the iteration message with data from the parent message or the current item.
func (n *ForEachNode) applyMapping(ctx types.NodeCtx, parentMsg, iterMsg types.RuleMsg, mappingConfig InputMappingConfig, item interface{}) error {
	var valueToSet interface{}
	var found bool
	var err error

	if mappingConfig.From == "_item" {
		if item == nil {
			return nil // Skip if item is nil (e.g., during pre-population)
		}
		valueToSet = item
		found = true
	} else {
		valueToSet, found, err = helper.ExtractFromMsgByPath(parentMsg, mappingConfig.From)
		if err != nil {
			return fmt.Errorf("failed to extract value from parent message with expression '%s': %w", mappingConfig.From, err)
		}
	}

	if !found {
		ctx.Warn("Source for input mapping not found, skipping.", "from", mappingConfig.From, "to", mappingConfig.To)
		return nil
	}

	// Determine destination and set value in iterMsg
	if strings.HasPrefix(mappingConfig.To, "dataT.") {
		objId := strings.TrimPrefix(mappingConfig.To, "dataT.")
		if objId == "" {
			return fmt.Errorf("invalid dataT mapping 'To' target: %s", mappingConfig.To)
		}
		// For SHARED mode, the object might already exist.
		coreObj, ok := iterMsg.DataT().Get(objId)
		if !ok {
			// If not, create it.
			if mappingConfig.DefineSID == "" {
				return fmt.Errorf("DefineSID is required for dataT mapping '%s'", mappingConfig.To)
			}
			coreObj, err = iterMsg.DataT().NewItem(mappingConfig.DefineSID, objId)
			if err != nil {
				return fmt.Errorf("failed to create new item with objId '%s' and sid '%s' in DataT: %w", objId, mappingConfig.DefineSID, err)
			}
		}
		coreObj.SetBody(valueToSet)

	} else if strings.HasPrefix(mappingConfig.To, "metadata.") {
		targetKey := strings.TrimPrefix(mappingConfig.To, "metadata.")
		if targetKey == "" {
			return fmt.Errorf("invalid metadata mapping 'To' target: %s", mappingConfig.To)
		}
		iterMsg.Metadata()[targetKey] = fmt.Sprintf("%v", valueToSet)

	} else if mappingConfig.To == "data" {
		if valueToSet != nil {
			if reflect.TypeOf(valueToSet).Kind() == reflect.Map || reflect.TypeOf(valueToSet).Kind() == reflect.Slice {
				jsonBytes, err := json.Marshal(valueToSet)
				if err != nil {
					return fmt.Errorf("failed to marshal data for 'data' mapping: %w", err)
				}
				iterMsg.SetData(string(jsonBytes))
			} else {
				iterMsg.SetData(fmt.Sprintf("%v", valueToSet))
			}
		}
	} else {
		return fmt.Errorf("unsupported input mapping 'To' target: %s", mappingConfig.To)
	}
	return nil
}

// executeSubChain adds metadata to the message and executes the sub-chain.
func (n *ForEachNode) executeSubChain(ctx types.NodeCtx, iterMsg types.RuleMsg, index, itemCount int, targetRuntime types.Runtime) (types.RuleMsg, error) {
	// Add loop metadata
	iterMsg.Metadata()[MetadataKeyLoopIndex] = fmt.Sprintf("%d", index)
	iterMsg.Metadata()[MetadataKeyIsLastItem] = fmt.Sprintf("%t", index == itemCount-1)

	// Execute synchronously and return the result
	return targetRuntime.ExecuteAndWait(ctx.GetContext(), "", iterMsg, nil)
}
