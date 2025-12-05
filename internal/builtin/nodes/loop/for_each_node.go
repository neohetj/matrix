package loop

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/NeohetJ/Matrix/internal/registry"
	"github.com/NeohetJ/Matrix/pkg/asset"
	"github.com/NeohetJ/Matrix/pkg/message"
	"github.com/NeohetJ/Matrix/pkg/types"
	"github.com/NeohetJ/Matrix/pkg/utils"
)

func init() {
	registry.Default.NodeManager.Register(forEachNodePrototype)
	registry.Default.FaultRegistry.Register(forEachNodePrototype.Errors()...)
}

const (
	ForEachNodeType       = "action/forEach"
	MetadataKeyLoopIndex  = "_loopIndex"
	MetadataKeyIsLastItem = "is_last_item"
)

var (
	FaultSourceNotFound = &types.Fault{
		Code:    "MATRIX_FOR_EACH_001",
		Message: "iteration source not found",
	}
	FaultInvalidType = &types.Fault{
		Code:    "MATRIX_FOR_EACH_002",
		Message: "invalid iteration source type",
	}
	FaultMappingFailed = &types.Fault{
		Code:    "MATRIX_FOR_EACH_003",
		Message: "iteration mapping failed",
	}
	FaultTargetChainNotFound = &types.Fault{
		Code:    "MATRIX_FOR_EACH_004",
		Message: "target chain not found",
	}
)

var forEachNodePrototype = &ForEachNode{
	BaseNode: *types.NewBaseNode(ForEachNodeType, types.NodeMetadata{
		Name:        "For Each",
		Description: "Iterates over a list or a range from msg.DataT or an expression, and executes a sub-chain for each item.",
		Dimension:   "Action",
		Tags:        []string{"action", "loop", "iterator"},
		Version:     "1.1.0",
		NodeWrites: []types.ContractDef{
			{URI: asset.MetadataURI(MetadataKeyLoopIndex), Description: "The 0-based index of the current loop iteration. This is added to the message sent to the sub-chain."},
			{URI: asset.MetadataURI(MetadataKeyIsLastItem), Description: "A boolean (as a string 'true' or 'false') indicating if it's the last item in the loop. This is added to the message sent to the sub-chain."},
		},
	}),
}

// ForEachNodeConfiguration holds the instance-specific configuration.
type ForEachNodeConfiguration struct {
	// LoopSource is an expression to get the list or count to iterate over.
	LoopSource string `json:"loopSource"`
	// Mode specifies the iteration mode: "LIST" (default) or "RANGE".
	Mode string `json:"mode,omitempty"`
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
	// InputMapping defines how to map data from the parent message to the sub-chain's message.
	InputMapping types.EndpointIOPacket `json:"inputMapping"`
	// OutputMapping defines how to map data from the sub-chain's result message back to the parent message.
	OutputMapping types.EndpointIOPacket `json:"outputMapping,omitempty"`
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

// GetInputMapping returns the configuration for mapping data from the parent message to the sub-chain message.
func (n *ForEachNode) GetInputMapping() types.EndpointIOPacket {
	return n.nodeConfig.InputMapping
}

// GetOutputMapping returns the configuration for mapping data from the sub-chain result back to the parent message.
func (n *ForEachNode) GetOutputMapping() types.EndpointIOPacket {
	return n.nodeConfig.OutputMapping
}

// GetTargetChainID returns the ID of the sub-chain being triggered.
func (n *ForEachNode) GetTargetChainID() string {
	return n.nodeConfig.ChainId
}

// Errors returns the list of possible faults for this node.
func (n *ForEachNode) Errors() []*types.Fault {
	return []*types.Fault{
		FaultSourceNotFound,
		FaultInvalidType,
		FaultMappingFailed,
		FaultTargetChainNotFound,
	}
}

// DataContract returns the data contract for the node based on configuration.
func (n *ForEachNode) DataContract() types.DataContract {
	// Base contract from metadata
	contract := n.BaseNode.DataContract()

	// 1. Loop Source (Read)
	if n.nodeConfig.LoopSource != "" {
		contract.Reads = append(contract.Reads, n.nodeConfig.LoopSource)
	}

	// 2. Input Mapping (Read from Parent)
	for _, field := range n.nodeConfig.InputMapping.Fields {
		if !strings.HasPrefix(field.Name, "_item") {
			contract.Reads = append(contract.Reads, field.Name)
		}
	}

	// 3. Output Mapping (Write to Parent)
	for _, field := range n.nodeConfig.OutputMapping.Fields {
		contract.Writes = append(contract.Writes, field.BindPath)
	}

	if n.nodeConfig.OutputMapping.MapAll != nil && *n.nodeConfig.OutputMapping.MapAll != "" {
		contract.Writes = append(contract.Writes, *n.nodeConfig.OutputMapping.MapAll)
	}

	return contract
}

// Init initializes the node instance.
func (n *ForEachNode) Init(configuration types.ConfigMap) error {
	if err := utils.Decode(configuration, &n.nodeConfig); err != nil {
		return fmt.Errorf("failed to decode forEach node config: %w", err)
	}

	if n.nodeConfig.ChainId == "" {
		return fmt.Errorf("'chainId' is not specified for node %s", n.ID())
	}
	if n.nodeConfig.LoopSource == "" {
		return fmt.Errorf("'loopSource' is not specified for node %s", n.ID())
	}

	// Default mode to LIST
	if n.nodeConfig.Mode == "" {
		n.nodeConfig.Mode = "LIST"
	}

	return nil
}

// OnMsg executes the loop.
func (n *ForEachNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {
	// 1. Get the target sub-chain runtime
	targetRuntime, ok := registry.Default.RuntimePool.Get(n.nodeConfig.ChainId)
	if !ok {
		ctx.HandleError(msg, FaultTargetChainNotFound.Wrap(fmt.Errorf("target chain with id '%s' not found", n.nodeConfig.ChainId)))
		return
	}

	// 2. Extract the source for iteration
	sourceRaw, err := message.ExtractFromMsg[any](msg, n.nodeConfig.LoopSource)
	if err != nil {
		ctx.HandleError(msg, FaultSourceNotFound.Wrap(fmt.Errorf("failed to extract iteration loopSource with expression '%s': %w", n.nodeConfig.LoopSource, err)))
		return
	}

	var itemCount int
	var itemsVal reflect.Value
	isRange := strings.ToUpper(n.nodeConfig.Mode) == "RANGE"

	if isRange {
		var fault *types.Fault
		itemCount, fault = n.getItemCount(sourceRaw)
		if fault != nil {
			ctx.HandleError(msg, fault)
			return
		}
	} else {
		var fault *types.Fault
		itemCount, itemsVal, fault = n.getItemsCountAndVal(sourceRaw)
		if fault != nil {
			ctx.HandleError(msg, fault)
			return
		}
	}

	// 3. Iterate and execute
	ctx.Info("Starting forEach loop", "chainId", n.nodeConfig.ChainId, "itemCount", itemCount, "mode", n.nodeConfig.Mode, "async", n.nodeConfig.Async, "messageScope", n.nodeConfig.MessageScope)

	if n.nodeConfig.Async {
		// Asynchronous execution (always uses INDEPENDENT message scope)
		var wg sync.WaitGroup
		for i := 0; i < itemCount; i++ {
			wg.Add(1)
			item := n.getCurrentItem(isRange, itemsVal, i)

			go func(index int, currentItem any) {
				defer wg.Done()
				iterMsg, fault := n.prepareIterMsg(ctx, msg, nil, currentItem)
				if fault != nil {
					ctx.Error("Async sub-chain mapping failed", "iteration", index, "error", fault)
					return
				}
				if _, err := n.executeSubChain(ctx, iterMsg, index, itemCount, targetRuntime); err != nil {
					ctx.Error("Async sub-chain execution failed", "iteration", index, "error", err)
				}
			}(i, item)
		}
		wg.Wait()
		ctx.TellSuccess(msg)
	} else {
		// Synchronous execution
		var sharedIterMsg types.RuleMsg
		if n.nodeConfig.MessageScope == "SHARED" {
			sharedIterMsg = message.NewSubMsg(msg, n.nodeConfig.ChainId)
			if fault := n.applyMappings(ctx, msg, sharedIterMsg, nil); fault != nil {
				ctx.HandleError(msg, fault)
				return
			}
		}

		for i := 0; i < itemCount; i++ {
			item := n.getCurrentItem(isRange, itemsVal, i)
			iterMsg, fault := n.prepareIterMsg(ctx, msg, sharedIterMsg, item)
			if fault != nil {
				ctx.HandleError(msg, fault)
				return
			}

			finalIterMsg, err := n.executeSubChain(ctx, iterMsg, i, itemCount, targetRuntime)
			if err != nil {
				ctx.Error("Sync sub-chain execution failed", "iteration", i, "error", err, "continueOnError", n.nodeConfig.ContinueOnError)
				if !n.nodeConfig.ContinueOnError {
					ctx.TellFailure(finalIterMsg, fmt.Errorf("forEach loop failed at iteration %d: %w", i, err))
					return
				}
				continue
			}
			if errMsg, ok := finalIterMsg.Metadata()[types.MetaError]; ok {
				ctx.Error("Sync sub-chain execution failed (metadata error)", "iteration", i, "error", errMsg, "continueOnError", n.nodeConfig.ContinueOnError)
				if !n.nodeConfig.ContinueOnError {
					ctx.TellFailure(finalIterMsg, fmt.Errorf("forEach loop failed at iteration %d: %s", i, errMsg))
					return
				}
				continue
			}

			// Check for break signal
			breakKey := fmt.Sprintf("%s_%s", MetadataKeyBreak, n.nodeConfig.ChainId)
			if breakVal, ok := finalIterMsg.Metadata()[breakKey]; ok && breakVal != "" {
				ctx.Info("Break signal received, stopping loop", "iteration", i, "breakKey", breakKey)
				// Clear the break signal to avoid polluting subsequent logic
				delete(finalIterMsg.Metadata(), breakKey)

				// Apply output mappings for the last iteration before breaking
				if fault := n.applyOutputMappings(ctx, finalIterMsg, msg); fault != nil {
					ctx.HandleError(msg, fault)
					return
				}
				break
			}

			// Apply output mappings from sub-chain result back to parent message
			if fault := n.applyOutputMappings(ctx, finalIterMsg, msg); fault != nil {
				ctx.HandleError(msg, fault)
				return
			}
		}
		ctx.Info("ForEach loop completed successfully")
		ctx.TellSuccess(msg)
	}
}

func (n *ForEachNode) prepareIterMsg(ctx types.NodeCtx, parentMsg, sharedMsg types.RuleMsg, item any) (types.RuleMsg, *types.Fault) {
	var iterMsg types.RuleMsg
	if sharedMsg != nil {
		iterMsg = sharedMsg
	} else {
		iterMsg = message.NewSubMsg(parentMsg, n.nodeConfig.ChainId)
	}

	if fault := n.applyMappings(ctx, parentMsg, iterMsg, item); fault != nil {
		return nil, fault
	}
	return iterMsg, nil
}

// applyMappings populates the iteration message with data from the parent message or the current item.
func (n *ForEachNode) applyMappings(ctx types.NodeCtx, parentMsg, iterMsg types.RuleMsg, item any) *types.Fault {
	mapping := n.nodeConfig.InputMapping

	// 1. Handle MapAll if present
	if mapping.MapAll != nil && *mapping.MapAll != "" {
		if item != nil {
			targetPath := *mapping.MapAll
			if err := message.SetInMsg(iterMsg, targetPath, item); err != nil {
				return FaultMappingFailed.Wrap(fmt.Errorf("failed to map entire item to '%s': %w", targetPath, err))
			}
		}
	}

	// 2. Handle individual Fields
	for _, field := range mapping.Fields {
		var valueToSet any
		var found bool
		var err error

		if strings.HasPrefix(field.Name, "_item") {
			if item == nil {
				continue // Skip if item is nil (e.g., during pre-population)
			}
			if field.Name == "_item" {
				valueToSet = item
				found = true
			} else if subPath, isSub := strings.CutPrefix(field.Name, "_item."); isSub {
				valueToSet, found = n.extractFromItem(item, subPath)
				if !found {
					ctx.Warn("Sub-field not found in item", "subPath", subPath)
				}
			}
		} else {
			// Extract from parent message using standard rulemsg URI
			valueToSet, err = message.ExtractFromMsg[any](parentMsg, field.Name)
			if err != nil {
				// If not found (and not an error in URI format), we might treat it as "not found"
				// But ExtractFromMsg currently returns error for "not found".
				// We need to differentiate "not found" from other errors?
				// For now, let's assume error means "failed to extract" (including not found).
				// We clear the error if we can use default value or if it's optional.
				// However, to keep logic consistent with "found" flag:
				found = false
			} else {
				found = true
			}
		}

		if !found {
			if field.Required {
				return FaultMappingFailed.Wrap(fmt.Errorf("source for required input mapping not found: %s", field.Name))
			}
			if field.DefaultValue != nil {
				valueToSet = field.DefaultValue
				found = true
			}
		}

		if found {
			if err := message.SetInMsg(iterMsg, field.BindPath, valueToSet); err != nil {
				return FaultMappingFailed.Wrap(fmt.Errorf("failed to set value in iteration message at '%s': %w", field.BindPath, err))
			}
		}
	}

	return nil
}

// applyOutputMappings maps data from the sub-chain result message back to the parent message.
func (n *ForEachNode) applyOutputMappings(ctx types.NodeCtx, subChainResultMsg, parentMsg types.RuleMsg) *types.Fault {
	mapping := n.nodeConfig.OutputMapping

	// 1. Handle MapAll if present
	if mapping.MapAll != nil && *mapping.MapAll != "" {
		// For output mapping, MapAll usually means taking the whole dataT from sub-chain and putting it somewhere in parent
		// But subChainResultMsg.DataT() is a DataT interface.
		// We might need to extract specific parts or the whole map.
		// Assuming MapAll targetPath in parentMsg.
		targetPath := *mapping.MapAll
		// We can't easily "get all" as a single object unless we know what it is.
		// But helper.ExtractFromMsg can handle "rulemsg://dataT".
		sourceVal, err := message.ExtractFromMsg[any](subChainResultMsg, "rulemsg://dataT")
		if err != nil {
			return FaultMappingFailed.Wrap(fmt.Errorf("failed to extract dataT from sub-chain result: %w", err))
		}
		if err := message.SetInMsg(parentMsg, targetPath, sourceVal); err != nil {
			return FaultMappingFailed.Wrap(fmt.Errorf("failed to map sub-chain result to '%s': %w", targetPath, err))
		}
	}

	// 2. Handle individual Fields
	for _, field := range mapping.Fields {
		// Extract from sub-chain result message
		valueToSet, err := message.ExtractFromMsg[any](subChainResultMsg, field.Name)
		found := err == nil

		if !found {
			if field.Required {
				return FaultMappingFailed.Wrap(fmt.Errorf("source for required output mapping not found: %s", field.Name))
			}
			if field.DefaultValue != nil {
				valueToSet = field.DefaultValue
				found = true
			}
		}

		if found {
			if err := message.SetInMsg(parentMsg, field.BindPath, valueToSet); err != nil {
				return FaultMappingFailed.Wrap(fmt.Errorf("failed to set value in parent message at '%s': %w", field.BindPath, err))
			}
		}
	}

	return nil
}

func (n *ForEachNode) extractFromItem(item any, path string) (any, bool) {
	// First try to treat item as a map
	if itemMap, ok := item.(map[string]any); ok {
		val, found := itemMap[path]
		return val, found
	}

	// Fallback to reflection for structs or other types
	val := reflect.ValueOf(item)
	if val.Kind() == reflect.Map {
		key := reflect.ValueOf(path)
		v := val.MapIndex(key)
		if v.IsValid() {
			return v.Interface(), true
		}
	}
	return nil, false
}

// executeSubChain adds metadata to the message and executes the sub-chain.
func (n *ForEachNode) executeSubChain(ctx types.NodeCtx, iterMsg types.RuleMsg, index, itemCount int, targetRuntime types.Runtime) (types.RuleMsg, error) {
	// Add loop metadata
	iterMsg.Metadata()[MetadataKeyLoopIndex] = fmt.Sprintf("%d", index)
	iterMsg.Metadata()[MetadataKeyIsLastItem] = fmt.Sprintf("%t", index == itemCount-1)

	// Execute synchronously and return the result
	return targetRuntime.ExecuteAndWait(ctx.GetContext(), "", iterMsg, nil)
}

func (n *ForEachNode) getCurrentItem(isRange bool, itemsVal reflect.Value, index int) any {
	if !isRange {
		return itemsVal.Index(index).Interface()
	}
	return index
}

func (n *ForEachNode) getItemCount(source any) (int, *types.Fault) {
	count, err := utils.RangeCount(source)
	if err != nil {
		return 0, FaultInvalidType.Wrap(err)
	}
	return count, nil
}

func (n *ForEachNode) getItemsCountAndVal(source any) (int, reflect.Value, *types.Fault) {
	itemsVal, err := utils.SliceValue(source)
	if err != nil {
		return 0, reflect.Value{}, FaultInvalidType.Wrap(err)
	}
	return itemsVal.Len(), itemsVal, nil
}
