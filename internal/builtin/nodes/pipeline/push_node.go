package pipeline

import (
	"fmt"
	"strings"
	"time"

	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/asset"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/helper"
	"github.com/neohetj/matrix/pkg/rulechain"
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
	// Channel push should only expose its local configuration dependencies.
	reads := make([]string, 0, 2)
	reads = append(reads, collectRuleMsgReadsFromConfigString(n.nodeConfig.PipelineID)...)
	reads = append(reads, collectRuleMsgReadsFromConfigString(n.nodeConfig.ChannelName)...)

	return types.DataContract{
		Reads: dedupeContractURIs(reads),
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

	msgCopy, err := n.cloneMsgForChannel(ctx, msg, pipelineID, channelName)
	if err != nil {
		ctx.HandleError(msg, fmt.Errorf("failed to project message for channel push: %w", err))
		return
	}

	// 4. Push to Channel
	if blocking {
		select {
		case ch <- msgCopy:
			ctx.TellSuccess(msg)
		case <-time.After(time.Duration(timeoutMs) * time.Millisecond):
			ctx.HandleError(msg, fmt.Errorf("%w: %s:%s", DefPushTimeout, pipelineID, channelName))
		case <-ctx.GetContext().Done():
			// Cancelled
		}
	} else {
		select {
		case ch <- msgCopy:
			ctx.TellSuccess(msg)
		default:
			ctx.HandleError(msg, DefChannelFull)
		}
	}
}

func (n *ChannelPushNode) cloneMsgForChannel(ctx types.NodeCtx, msg types.RuleMsg, pipelineID string, channelName string) (types.RuleMsg, error) {
	requiredInputs, resolved := n.resolveChannelRequiredInputs(ctx, pipelineID, channelName)
	if !resolved || requiredInputs.RetainAll {
		return msg.DeepCopy()
	}
	if msg.DataT() == nil {
		return msg.DeepCopy()
	}
	projectedDataT, err := msg.DataT().Project(requiredInputs.ObjIDs)
	if err != nil {
		return nil, err
	}
	if types.CloneMsgWithDataT == nil {
		return msg.DeepCopy()
	}
	return types.CloneMsgWithDataT(msg, projectedDataT), nil
}

func (n *ChannelPushNode) resolveChannelRequiredInputs(ctx types.NodeCtx, pipelineID string, channelName string) (types.CoreObjSet, bool) {
	runtime := ctx.GetRuntime()
	if runtime == nil || runtime.GetEngine() == nil {
		return types.CoreObjSet{RetainAll: true}, false
	}
	sharedPool := runtime.GetEngine().SharedNodePool()
	if sharedPool == nil {
		return types.CoreObjSet{RetainAll: true}, false
	}
	sharedCtx, ok := sharedPool.Get(pipelineID)
	if !ok || sharedCtx == nil {
		return types.CoreObjSet{RetainAll: true}, false
	}
	router, ok := sharedCtx.GetNode().(types.PipelineInputRouter)
	if !ok {
		return types.CoreObjSet{RetainAll: true}, false
	}
	targetChainIDs := router.GetTargetChainIDsForInputChannel(channelName)
	if len(targetChainIDs) == 0 {
		return types.CoreObjSet{RetainAll: true}, false
	}
	runtimePool := runtime.GetEngine().RuntimePool()
	if runtimePool == nil {
		return types.CoreObjSet{RetainAll: true}, false
	}

	requiredInputs := types.CoreObjSet{}
	for _, targetChainID := range targetChainIDs {
		targetRuntime, ok := runtimePool.Get(targetChainID)
		if !ok || targetRuntime == nil {
			return types.CoreObjSet{RetainAll: true}, false
		}
		requiredInputs = unionCoreObjSets(requiredInputs, rulechain.ResolveRequiredInputs(targetRuntime))
		if requiredInputs.RetainAll {
			return requiredInputs, true
		}
	}
	return requiredInputs, true
}

func unionCoreObjSets(base types.CoreObjSet, other types.CoreObjSet) types.CoreObjSet {
	if base.RetainAll || other.RetainAll {
		return types.CoreObjSet{RetainAll: true}
	}
	seen := make(map[string]struct{}, len(base.ObjIDs)+len(other.ObjIDs))
	objIDs := make([]string, 0, len(base.ObjIDs)+len(other.ObjIDs))
	for _, objID := range base.ObjIDs {
		objID = strings.TrimSpace(objID)
		if objID == "" {
			continue
		}
		if _, ok := seen[objID]; ok {
			continue
		}
		seen[objID] = struct{}{}
		objIDs = append(objIDs, objID)
	}
	for _, objID := range other.ObjIDs {
		objID = strings.TrimSpace(objID)
		if objID == "" {
			continue
		}
		if _, ok := seen[objID]; ok {
			continue
		}
		seen[objID] = struct{}{}
		objIDs = append(objIDs, objID)
	}
	return types.CoreObjSet{ObjIDs: objIDs}
}
