package pipeline

import (
	"context"
	"fmt"
	"sync"

	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/asset"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/pkg/utils"
)

const (
	PipelineEndpointNodeType = "endpoint/pipeline"
)

type ProcessorConfig struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type PipelineStageConfig struct {
	Name          string          `json:"name"`
	ID            string          `json:"id"`
	Concurrency   int             `json:"concurrency"`
	BufferSize    int             `json:"bufferSize"`
	Processor     ProcessorConfig `json:"processor"` // Reference to a chain to execute
	InputChannel  string          `json:"inputChannel,omitempty"`
	OutputChannel string          `json:"outputChannel,omitempty"`
}

type PipelineConfig struct {
	Stages          []PipelineStageConfig `json:"stages"`
	ExposedChannels map[string]string     `json:"exposedChannels"` // "input": "stage_id_in"
	// ChannelManager is the URI reference to the shared channel manager node (e.g. ref://channel_manager)
	ChannelManager string `json:"channelManager" description:"URI reference to the shared channel manager node"`
}

// pipelineEndpointNodePrototype is the shared prototype instance used for registration.
var pipelineEndpointNodePrototype = &PipelineEndpointNode{
	BaseNode: *types.NewBaseNode(PipelineEndpointNodeType, types.NodeMetadata{
		Name:        "Pipeline Endpoint",
		Description: "Orchestrates concurrent processing stages connected by channels.",
		Dimension:   "Endpoint",
		Tags:        []string{"endpoint", "pipeline", "concurrency"},
		Version:     "1.0.0",
	}),
}

func init() {
	registry.Default.GetNodeManager().Register(pipelineEndpointNodePrototype)
}

type PipelineEndpointNode struct {
	types.BaseNode
	types.Instance
	config      PipelineConfig
	runtimePool types.RuntimePool
	// Active channels managed by this endpoint instance
	activeChannels map[string]chan types.RuleMsg
	cancelFunc     context.CancelFunc
	mu             sync.Mutex
	channelManager *ChannelManager
}

// Ensure PipelineEndpointNode implements ActiveEndpoint interface
var _ types.ActiveEndpoint = (*PipelineEndpointNode)(nil)

// OnMsg implements the Node interface.
// While PipelineEndpoint is primarily an Endpoint, implementing OnMsg allows it to be used
// as a regular node in a chain, potentially to trigger the pipeline or perform setup.
// Currently, it's a no-op or can be used to re-trigger/re-configure if needed.
func (n *PipelineEndpointNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {
	// If used as a node, we might want to push the message to an "input" channel if configured?
	// Or simply treat it as a pass-through.
	// For now, we'll just log and pass success.
	// This ensures compatibility if the node is placed in a flow.
	ctx.TellSuccess(msg)
}

func (n *PipelineEndpointNode) New() types.Node {
	return &PipelineEndpointNode{BaseNode: n.BaseNode}
}

func (n *PipelineEndpointNode) Init(config types.ConfigMap) error {
	if err := utils.Decode(config, &n.config); err != nil {
		return types.InvalidConfiguration.Wrap(err)
	}
	n.activeChannels = make(map[string]chan types.RuleMsg)
	return nil
}

// SetRuntimePool implements the types.Endpoint interface
func (n *PipelineEndpointNode) SetRuntimePool(pool any) error {
	if p, ok := pool.(types.RuntimePool); ok {
		n.runtimePool = p
		return nil
	}
	return types.InvalidConfiguration
}

// Start initializes the pipeline, creating channels and workers
func (n *PipelineEndpointNode) Start(ctx context.Context) error {
	ctx, n.cancelFunc = context.WithCancel(ctx)

	// Resolve Channel Manager
	if n.config.ChannelManager != "" {
		pool := registry.Default.GetSharedNodePool()
		ctx := asset.NewAssetContext(asset.WithNodePool(pool))
		ast := asset.Asset[*ChannelManager]{URI: n.config.ChannelManager}
		cm, err := ast.Resolve(ctx)
		if err != nil {
			return fmt.Errorf("failed to resolve channel manager '%s': %w", n.config.ChannelManager, err)
		}
		n.channelManager = cm
	} else {
		return fmt.Errorf("channelManagerId is required")
	}

	// 1. Create Channels
	// Initialize channels defined in ExposedChannels.
	for logicalName, channelKey := range n.config.ExposedChannels {
		ch := make(chan types.RuleMsg, 100) // Buffer size could be configurable
		n.activeChannels[channelKey] = ch
		n.channelManager.Register(n.ID(), logicalName, ch)
		// Also register the internal key if it differs, so it can be accessed by its ID
		if logicalName != channelKey {
			n.channelManager.Register(n.ID(), channelKey, ch)
		}
	}

	// 2. Start Workers for each Stage
	for _, stage := range n.config.Stages {
		go n.startStageWorkers(ctx, stage)
	}

	return nil
}

func (n *PipelineEndpointNode) startStageWorkers(ctx context.Context, stage PipelineStageConfig) {
	chName := stage.InputChannel
	if chName == "" {
		// Skip if no input channel defined (e.g. generator stage)
		return
	}

	// Ensure channel exists
	n.mu.Lock()
	ch, ok := n.activeChannels[chName]
	if !ok {
		ch = make(chan types.RuleMsg, 100)
		n.activeChannels[chName] = ch
		// Register globally so push nodes can find it
		n.channelManager.Register(n.ID(), chName, ch)
	}
	n.mu.Unlock()

	// Also register aliases
	for alias, realName := range n.config.ExposedChannels {
		if realName == chName {
			n.channelManager.Register(n.ID(), alias, ch)
		}
	}

	for i := 0; i < stage.Concurrency; i++ {
		go func(workerID int) {
			for {
				select {
				case <-ctx.Done():
					return
				case msg := <-ch:
					n.processData(ctx, stage, msg)
				}
			}
		}(i)
	}
}

func (n *PipelineEndpointNode) processData(ctx context.Context, stage PipelineStageConfig, msg types.RuleMsg) {
	// Execute the processor.
	// stage.Processor.ID is a RuleChain ID.
	rt, ok := n.runtimePool.Get(stage.Processor.ID)
	if !ok {
		// Fallback to global pool
		rt, ok = registry.Default.RuntimePool.Get(stage.Processor.ID)
		if !ok {
			fmt.Printf("Error: Runtime not found for processor %s in stage %s\n", stage.Processor.ID, stage.Name)
			return
		}
	}

	// Create new msg with same Data and Metadata
	newMsg := types.NewMsg(stage.Processor.ID, string(msg.Data()), msg.Metadata(), msg.DataT())

	// Execute from default start node
	err := rt.Execute(ctx, "", newMsg, func(result types.RuleMsg, err error) {
		if err != nil {
			fmt.Printf("Stage %s processing error: %v\n", stage.Name, err)
			return
		}
		// Output handling
		// If stage has an output channel, push the result there.
		if stage.OutputChannel != "" {
			resCopy, err := result.DeepCopy()
			if err != nil {
				fmt.Printf("Stage %s deep copy error: %v\n", stage.Name, err)
				return
			}
			n.pushToChannel(stage.OutputChannel, resCopy)
		}
	})

	if err != nil {
		fmt.Printf("Stage %s execution start error: %v\n", stage.Name, err)
	}
}

func (n *PipelineEndpointNode) pushToChannel(channelName string, msg types.RuleMsg) {
	n.mu.Lock()
	ch, ok := n.activeChannels[channelName]
	if !ok {
		// Create if not exists (lazy)
		ch = make(chan types.RuleMsg, 100)
		n.activeChannels[channelName] = ch
		n.channelManager.Register(n.ID(), channelName, ch)
	}
	n.mu.Unlock()

	// Non-blocking push to avoid deadlocks in this simple impl
	select {
	case ch <- msg:
	default:
		fmt.Printf("Warning: Channel %s full, dropping message\n", channelName)
	}
}

func (n *PipelineEndpointNode) Stop() error {
	if n.cancelFunc != nil {
		n.cancelFunc()
	}
	// Unregister channels
	if n.channelManager != nil {
		for name := range n.activeChannels {
			n.channelManager.Unregister(n.ID(), name)
		}
		for alias := range n.config.ExposedChannels {
			n.channelManager.Unregister(n.ID(), alias)
		}
	}
	return nil
}

// Extend Stage Config with Input/Output channels as per Section 2.2
func (n *PipelineEndpointNode) GetInstance() (any, error) {
	return n, nil
}

// GetTargetChainIDs implements the types.MultiChainTrigger interface.
func (n *PipelineEndpointNode) GetTargetChainIDs() []string {
	var ids []string
	seen := make(map[string]struct{})
	for _, stage := range n.config.Stages {
		if stage.Processor.ID != "" {
			if _, exists := seen[stage.Processor.ID]; !exists {
				ids = append(ids, stage.Processor.ID)
				seen[stage.Processor.ID] = struct{}{}
			}
		}
	}
	return ids
}
