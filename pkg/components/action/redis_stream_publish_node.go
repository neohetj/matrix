package action

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/asset"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/helper"
	"github.com/neohetj/matrix/pkg/message"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/pkg/utils"
	"github.com/redis/go-redis/v9"
)

const (
	RedisStreamPublishNodeType = "action/redis_stream_publish"
)

var (
	FaultRedisStreamPublishConfig = &types.Fault{Code: cnst.CodeRedisStreamPublishConfig, Message: "invalid redis stream publish configuration"}
	FaultRedisStreamPublishFailed = &types.Fault{Code: cnst.CodeRedisStreamPublishFailed, Message: "redis stream publish failed"}
)

var redisStreamPublishNodePrototype = &RedisStreamPublishNode{
	BaseNode: *types.NewBaseNode(RedisStreamPublishNodeType, types.NodeMetadata{
		Name:        "Redis Stream Publish",
		Description: "Publishes a mapped RuleMsg payload to a Redis Stream.",
		Dimension:   "Action",
		Tags:        []string{"action", "redis", "stream", "event", "publish"},
		Version:     "1.0.0",
		Icon:        "message-queue",
	}),
}

func init() {
	registry.Default.NodeManager.Register(redisStreamPublishNodePrototype)
	registry.Default.FaultRegistry.Register(FaultRedisStreamPublishConfig, FaultRedisStreamPublishFailed)
}

type RedisStreamPublishNodeConfiguration struct {
	RedisClient string                 `json:"redisClient"`
	Stream      string                 `json:"stream"`
	Values      types.EndpointIOPacket `json:"values,omitempty"`
	MaxLen      int64                  `json:"maxLen,omitempty"`
	Approx      bool                   `json:"approx,omitempty"`
	IDTarget    string                 `json:"idTarget,omitempty"`
}

type RedisStreamPublishNode struct {
	types.BaseNode
	types.Instance
	nodeConfig RedisStreamPublishNodeConfiguration
}

func (n *RedisStreamPublishNode) New() types.Node {
	return &RedisStreamPublishNode{BaseNode: n.BaseNode}
}

func (n *RedisStreamPublishNode) Init(cfg types.ConfigMap) error {
	if err := utils.Decode(cfg, &n.nodeConfig); err != nil {
		return FaultRedisStreamPublishConfig.Wrap(err)
	}
	if strings.TrimSpace(n.nodeConfig.RedisClient) == "" {
		return FaultRedisStreamPublishConfig.Wrap(fmt.Errorf("redisClient is required"))
	}
	if strings.TrimSpace(n.nodeConfig.Stream) == "" {
		return FaultRedisStreamPublishConfig.Wrap(fmt.Errorf("stream is required"))
	}
	return nil
}

func (n *RedisStreamPublishNode) DataContract() types.DataContract {
	reads := make([]string, 0, len(n.nodeConfig.Values.Fields)+1)
	if n.nodeConfig.Values.MapAll != nil && strings.TrimSpace(*n.nodeConfig.Values.MapAll) != "" {
		reads = append(reads, strings.TrimSpace(*n.nodeConfig.Values.MapAll))
	}
	for _, field := range n.nodeConfig.Values.Fields {
		if strings.TrimSpace(field.BindPath) != "" {
			reads = append(reads, strings.TrimSpace(field.BindPath))
		}
	}
	writes := []string{}
	if strings.TrimSpace(n.nodeConfig.IDTarget) != "" {
		writes = append(writes, strings.TrimSpace(n.nodeConfig.IDTarget))
	}
	return types.DataContract{Reads: reads, Writes: writes}
}

func (n *RedisStreamPublishNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {
	client, err := resolveRedisClient(ctx, n.nodeConfig.RedisClient)
	if err != nil {
		ctx.HandleError(msg, FaultRedisStreamPublishFailed.Wrap(err))
		return
	}

	stream, err := renderRedisStreamTemplate(ctx, msg, n.nodeConfig.Stream)
	if err != nil {
		ctx.HandleError(msg, FaultRedisStreamPublishFailed.Wrap(err))
		return
	}
	stream = strings.TrimSpace(stream)
	if stream == "" {
		ctx.HandleError(msg, FaultRedisStreamPublishConfig.Wrap(fmt.Errorf("stream is empty")))
		return
	}

	values, err := n.buildValues(ctx, msg)
	if err != nil {
		ctx.HandleError(msg, FaultRedisStreamPublishFailed.Wrap(err))
		return
	}

	id, err := client.XAdd(ctx.GetContext(), &redis.XAddArgs{
		Stream: stream,
		MaxLen: n.nodeConfig.MaxLen,
		Approx: n.nodeConfig.Approx,
		Values: values,
	}).Result()
	if err != nil {
		ctx.HandleError(msg, FaultRedisStreamPublishFailed.Wrap(err))
		return
	}

	if target := strings.TrimSpace(n.nodeConfig.IDTarget); target != "" {
		if err := message.SetInMsg(msg, target, id); err != nil {
			ctx.HandleError(msg, FaultRedisStreamPublishFailed.Wrap(err))
			return
		}
	}

	ctx.Info("redis stream message published", "stream", stream, "message_id", id)
	ctx.TellSuccess(msg)
}

func (n *RedisStreamPublishNode) buildValues(ctx types.NodeCtx, msg types.RuleMsg) (map[string]any, error) {
	if n.nodeConfig.Values.MapAll == nil && len(n.nodeConfig.Values.Fields) == 0 {
		return map[string]any{
			"message_id":   msg.ID(),
			"message_type": msg.Type(),
			"payload":      string(msg.Data()),
		}, nil
	}
	raw, err := helper.ProcessOutbound(ctx, msg, n.nodeConfig.Values, helper.RuleMsgProvider{Msg: msg})
	if err != nil {
		return nil, err
	}
	values, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("redis stream values must evaluate to an object, got %T", raw)
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out[key] = normalizeRedisStreamValue(value)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("redis stream values are empty")
	}
	return out, nil
}

func normalizeRedisStreamValue(value any) any {
	switch v := value.(type) {
	case nil, string, []byte, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		return v
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(data)
	}
}

func resolveRedisClient(ctx types.NodeCtx, redisClientURI string) (*redis.Client, error) {
	redisClientURI = strings.TrimSpace(redisClientURI)
	if redisClientURI == "" {
		return nil, fmt.Errorf("redisClient is required")
	}

	var pool types.NodePool
	if ctx != nil && ctx.GetRuntime() != nil && ctx.GetRuntime().GetEngine() != nil {
		pool = ctx.GetRuntime().GetEngine().SharedNodePool()
	}
	if pool == nil {
		pool = registry.Default.GetSharedNodePool()
	}

	ast := asset.Asset[*redis.Client]{URI: redisClientURI}
	return ast.Resolve(asset.NewAssetContext(asset.WithNodePool(pool)))
}

func renderRedisStreamTemplate(ctx types.NodeCtx, msg types.RuleMsg, raw string) (string, error) {
	renderCtx := asset.NewAssetContext(asset.WithNodeCtx(ctx), asset.WithRuleMsg(msg))
	return asset.RenderTemplate(raw, renderCtx)
}
