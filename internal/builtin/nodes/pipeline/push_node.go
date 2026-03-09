package pipeline

import (
	"fmt"
	"strings"
	"time"

	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/asset"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/helper"
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

func (n *ChannelPushNode) DataContract() types.DataContract {
	// Keep pass-through semantics, and additionally expose explicit rulemsg dependencies
	// when pipelineId/channelName are configured from rulemsg template placeholders.
	reads := []string{"rulemsg://*"}
	reads = append(reads, collectRuleMsgReadsFromConfigString(n.nodeConfig.PipelineID)...)
	reads = append(reads, collectRuleMsgReadsFromConfigString(n.nodeConfig.ChannelName)...)

	return types.DataContract{
		Reads:  dedupeContractURIs(reads),
		Writes: []string{"rulemsg://*"},
	}
}

func collectRuleMsgReadsFromConfigString(raw string) []string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}

	uris := make([]string, 0)
	if asset.IsTemplate(value) {
		uris = asset.CollectTemplateAssets(value)
	} else {
		uris = []string{asset.NormalizeURI(value)}
	}

	result := make([]string, 0, len(uris))
	for _, uri := range uris {
		uri = strings.TrimSpace(uri)
		if strings.HasPrefix(uri, "rulemsg://") {
			result = append(result, uri)
		}
	}
	return result
}

func dedupeContractURIs(uris []string) []string {
	if len(uris) == 0 {
		return nil
	}
	result := make([]string, 0, len(uris))
	seen := make(map[string]struct{}, len(uris))
	for _, uri := range uris {
		uri = strings.TrimSpace(uri)
		if uri == "" {
			continue
		}
		if _, ok := seen[uri]; ok {
			continue
		}
		seen[uri] = struct{}{}
		result = append(result, uri)
	}
	return result
}

func (n *ChannelPushNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {
	// 1. Get Config (support template rendering)
	pipelineID := n.nodeConfig.PipelineID
	channelName := n.nodeConfig.ChannelName
	blocking := n.nodeConfig.Blocking
	timeoutMs := n.nodeConfig.Timeout

	assetCtx := asset.NewAssetContext(
		asset.WithNodeCtx(ctx),
		asset.WithRuleMsg(msg),
	)

	if rendered, err := asset.RenderTemplate(pipelineID, assetCtx); err == nil {
		if v, vErr := helper.RenderAsset[string](rendered); vErr == nil && v != "" {
			pipelineID = v
		}
	}
	if rendered, err := asset.RenderTemplate(channelName, assetCtx); err == nil {
		if v, vErr := helper.RenderAsset[string](rendered); vErr == nil && v != "" {
			channelName = v
		}
	}

	if pipelineID == "" || channelName == "" {
		ctx.HandleError(msg, fmt.Errorf("pipelineId/channelName is empty"))
		return
	}

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
