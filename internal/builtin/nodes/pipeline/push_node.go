package pipeline

import (
	"fmt"
	"time"

	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/asset"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/pkg/utils"
)

const (
	ChannelPushNodeType = "action/channel_push"

	CfgPipelineID  = "pipelineId"
	CfgChannelName = "channelName"
	CfgBlocking    = "blocking"
	CfgTimeout     = "timeout"
	ParamData      = "data"
)

var (
	DefChannelNotFound = &types.Fault{Code: cnst.CodePipelineChannelNotFound, Message: "channel not found"}
	DefPushTimeout     = &types.Fault{Code: cnst.CodePipelinePushTimeout, Message: "timeout pushing to channel"}
	DefChannelFull     = &types.Fault{Code: cnst.CodePipelineChannelFull, Message: "channel full"}
)

var channelPushNodePrototype = &ChannelPushNode{
	BaseNode: *types.NewBaseNode(ChannelPushNodeType, types.NodeMetadata{
		Name:        "Channel Push",
		Description: "Pushes data to a named channel in a pipeline.",
		Dimension:   "Action",
		Tags:        []string{"action", "channel", "push", "pipeline"},
		Version:     "1.0.0",
	}),
}

func init() {
	registry.Default.GetNodeManager().Register(channelPushNodePrototype)
	registry.Default.GetFaultRegistry().Register(
		DefChannelNotFound,
		DefPushTimeout,
		DefChannelFull,
	)
}

type ChannelPushNodeConfiguration struct {
	PipelineID  string `json:"pipelineId"`
	ChannelName string `json:"channelName"`
	Blocking    bool   `json:"blocking"`
	Timeout     int    `json:"timeout"`
	// ChannelManager is the URI reference to the shared channel manager node (e.g. ref://channel_manager)
	ChannelManager string `json:"channelManager" description:"URI reference to the shared channel manager node"`
}

type ChannelPushNode struct {
	types.BaseNode
	types.Instance
	nodeConfig ChannelPushNodeConfiguration
}

func (n *ChannelPushNode) New() types.Node {
	return &ChannelPushNode{BaseNode: n.BaseNode}
}

func (n *ChannelPushNode) Init(config types.ConfigMap) error {
	if err := utils.Decode(config, &n.nodeConfig); err != nil {
		return types.InvalidConfiguration.Wrap(err)
	}
	if n.nodeConfig.Timeout == 0 {
		n.nodeConfig.Timeout = 5000
	}
	return nil
}

func (n *ChannelPushNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {
	// 1. Get Config (prefer dynamic over static if needed, but here we use static mostly)
	// For full dynamic support, we could still use helper.GetConfigAsset if keys are expressions.
	// But assuming simple config for now as per init.
	pipelineID := n.nodeConfig.PipelineID
	channelName := n.nodeConfig.ChannelName
	blocking := n.nodeConfig.Blocking
	timeoutMs := n.nodeConfig.Timeout

	// If pipelineID/channelName are empty, maybe try dynamic lookup?
	// The original code used GetConfigAsset which supports template rendering.
	// If we want to support templates in these fields, we should use RenderConfigAsset.
	// But if we want to follow standard pattern, we use the struct.
	// Let's stick to the struct for performance, unless dynamic behavior is explicitly required.
	// The design doc example showed simple strings.

	// 3. Resolve Channel
	cmURI := n.nodeConfig.ChannelManager
	if cmURI == "" {
		ctx.HandleError(msg, fmt.Errorf("channelManager is required"))
		return
	}

	ast := asset.Asset[*ChannelManager]{URI: cmURI}
	pool := registry.Default.GetSharedNodePool()
	resolveCtx := asset.NewAssetContext(asset.WithNodePool(pool))

	cm, err := ast.Resolve(resolveCtx)
	if err != nil {
		ctx.HandleError(msg, fmt.Errorf("failed to resolve channel manager: %w", err))
		return
	}

	ch, err := cm.Get(pipelineID, channelName)
	if err != nil {
		ctx.HandleError(msg, fmt.Errorf("%w: %v", DefChannelNotFound, err))
		return
	}

	// 4. Push to Channel
	if blocking {
		// We MUST DeepCopy the message because it's crossing goroutine boundaries (channel).
		// The original message (and DataT) might be modified by subsequent nodes.
		msgCopy, err := msg.DeepCopy()
		if err != nil {
			ctx.HandleError(msg, fmt.Errorf("failed to deep copy message: %w", err))
			return
		}

		select {
		case ch <- msgCopy:
			ctx.TellSuccess(msg)
		case <-time.After(time.Duration(timeoutMs) * time.Millisecond):
			ctx.HandleError(msg, fmt.Errorf("%w: %s:%s", DefPushTimeout, pipelineID, channelName))
		case <-ctx.GetContext().Done():
			// Cancelled
		}
	} else {
		// We MUST DeepCopy the message because it's crossing goroutine boundaries (channel).
		msgCopy, err := msg.DeepCopy()
		if err != nil {
			ctx.HandleError(msg, fmt.Errorf("failed to deep copy message: %w", err))
			return
		}

		select {
		case ch <- msgCopy:
			ctx.TellSuccess(msg)
		default:
			ctx.HandleError(msg, DefChannelFull)
		}
	}
}
