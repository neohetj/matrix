package endpoint

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/asset"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/helper"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/pkg/utils"
	"github.com/redis/go-redis/v9"
)

const (
	RedisStreamEndpointNodeType = "endpoint/redis_stream"
)

var (
	FaultRedisStreamEndpointConfig  = &types.Fault{Code: cnst.CodeRedisStreamEndpointConfig, Message: "invalid redis stream endpoint configuration"}
	FaultRedisStreamEndpointRuntime = &types.Fault{Code: cnst.CodeRedisStreamEndpointRuntime, Message: "redis stream endpoint runtime failed"}
)

var redisStreamEndpointNodePrototype = &RedisStreamEndpointNode{
	BaseNode: *types.NewBaseNode(RedisStreamEndpointNodeType, types.NodeMetadata{
		Name:        "Redis Stream Endpoint",
		Description: "Consumes Redis Stream messages with a consumer group and triggers a rule chain.",
		Dimension:   "Endpoint",
		Tags:        []string{"endpoint", "redis", "stream", "event", "consumer"},
		Version:     "1.0.0",
		Icon:        "message-queue",
	}),
}

func init() {
	registry.Default.GetNodeManager().Register(redisStreamEndpointNodePrototype)
	registry.Default.GetFaultRegistry().Register(FaultRedisStreamEndpointConfig, FaultRedisStreamEndpointRuntime)
}

type RedisStreamEndpointConfiguration struct {
	RedisClient     string                 `json:"redisClient"`
	Stream          string                 `json:"stream"`
	Group           string                 `json:"group"`
	Consumer        string                 `json:"consumer,omitempty"`
	RuleChainID     string                 `json:"ruleChainId"`
	StartNodeID     string                 `json:"startNodeId,omitempty"`
	Input           types.EndpointIOPacket `json:"input,omitempty"`
	Count           int64                  `json:"count,omitempty"`
	BlockMs         int64                  `json:"blockMs,omitempty"`
	Concurrency     int                    `json:"concurrency,omitempty"`
	AutoCreateGroup bool                   `json:"autoCreateGroup,omitempty"`
	GroupStartID    string                 `json:"groupStartId,omitempty"`
	AckOnFailure    bool                   `json:"ackOnFailure,omitempty"`
}

type RedisStreamEndpointNode struct {
	types.BaseNode
	types.Instance
	config      RedisStreamEndpointConfiguration
	runtimePool types.RuntimePool
	client      *redis.Client
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	startOnce   sync.Once
	stopOnce    sync.Once
	startErr    error
}

var _ types.ActiveEndpoint = (*RedisStreamEndpointNode)(nil)
var _ types.SubChainTrigger = (*RedisStreamEndpointNode)(nil)

func (n *RedisStreamEndpointNode) New() types.Node {
	return &RedisStreamEndpointNode{BaseNode: n.BaseNode}
}

func (n *RedisStreamEndpointNode) Init(cfg types.ConfigMap) error {
	if err := utils.Decode(cfg, &n.config); err != nil {
		return FaultRedisStreamEndpointConfig.Wrap(err)
	}
	if strings.TrimSpace(n.config.RedisClient) == "" {
		return FaultRedisStreamEndpointConfig.Wrap(fmt.Errorf("redisClient is required"))
	}
	if strings.TrimSpace(n.config.Stream) == "" {
		return FaultRedisStreamEndpointConfig.Wrap(fmt.Errorf("stream is required"))
	}
	if strings.TrimSpace(n.config.Group) == "" {
		return FaultRedisStreamEndpointConfig.Wrap(fmt.Errorf("group is required"))
	}
	if strings.TrimSpace(n.config.RuleChainID) == "" {
		return FaultRedisStreamEndpointConfig.Wrap(fmt.Errorf("ruleChainId is required"))
	}
	if n.config.Count <= 0 {
		n.config.Count = 10
	}
	if n.config.BlockMs <= 0 {
		n.config.BlockMs = 5000
	}
	if n.config.Concurrency <= 0 {
		n.config.Concurrency = 1
	}
	if strings.TrimSpace(n.config.GroupStartID) == "" {
		n.config.GroupStartID = "0"
	}
	return nil
}

func (n *RedisStreamEndpointNode) SetRuntimePool(pool any) error {
	if p, ok := pool.(types.RuntimePool); ok {
		n.runtimePool = p
		return nil
	}
	return types.InvalidConfiguration
}

func (n *RedisStreamEndpointNode) Start(ctx context.Context) error {
	n.startOnce.Do(func() {
		client, err := n.resolveClient()
		if err != nil {
			n.startErr = err
			return
		}
		n.client = client
		n.ctx, n.cancel = context.WithCancel(ctx)
		if n.config.AutoCreateGroup {
			if err := n.ensureGroup(n.ctx); err != nil {
				n.startErr = err
				return
			}
		}
		for i := 0; i < n.config.Concurrency; i++ {
			n.wg.Add(1)
			go n.runWorker(i)
		}
	})
	return n.startErr
}

func (n *RedisStreamEndpointNode) Stop() error {
	n.stopOnce.Do(func() {
		if n.cancel != nil {
			n.cancel()
		}
		n.wg.Wait()
	})
	return nil
}

func (n *RedisStreamEndpointNode) GetInstance() (any, error) {
	return n, nil
}

func (n *RedisStreamEndpointNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {
	ctx.TellSuccess(msg)
}

func (n *RedisStreamEndpointNode) GetInputMapping() types.EndpointIOPacket {
	return n.config.Input
}

func (n *RedisStreamEndpointNode) GetOutputMapping() types.EndpointIOPacket {
	return types.EndpointIOPacket{}
}

func (n *RedisStreamEndpointNode) GetTargetChainID() string {
	return n.config.RuleChainID
}

func (n *RedisStreamEndpointNode) runWorker(workerID int) {
	defer n.wg.Done()
	consumer := n.consumerName(workerID)
	for {
		select {
		case <-n.ctx.Done():
			return
		default:
		}

		streams, err := n.client.XReadGroup(n.ctx, &redis.XReadGroupArgs{
			Group:    n.config.Group,
			Consumer: consumer,
			Streams:  []string{n.config.Stream, ">"},
			Count:    n.config.Count,
			Block:    time.Duration(n.config.BlockMs) * time.Millisecond,
		}).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				continue
			}
			fmt.Printf("redis stream endpoint read error: stream=%s group=%s consumer=%s error=%v\n", n.config.Stream, n.config.Group, consumer, err)
			time.Sleep(time.Second)
			continue
		}
		for _, stream := range streams {
			for _, redisMessage := range stream.Messages {
				n.handleMessage(n.ctx, stream.Stream, redisMessage)
			}
		}
	}
}

func (n *RedisStreamEndpointNode) handleMessage(ctx context.Context, stream string, redisMessage redis.XMessage) {
	rt, ok := n.resolveRuntime()
	if !ok {
		fmt.Printf("redis stream endpoint runtime not found: ruleChainId=%s\n", n.config.RuleChainID)
		return
	}
	msg, err := n.buildRuleMsg(stream, redisMessage)
	if err != nil {
		fmt.Printf("redis stream endpoint build message failed: stream=%s message_id=%s error=%v\n", stream, redisMessage.ID, err)
		if n.config.AckOnFailure {
			_, _ = n.client.XAck(ctx, stream, n.config.Group, redisMessage.ID).Result()
		}
		return
	}
	finalMsg, err := rt.ExecuteAndWait(ctx, n.config.StartNodeID, msg, nil)
	if err == nil && finalMsg != nil {
		if errText, failed := finalMsg.Metadata()[types.MetaError]; failed && strings.TrimSpace(errText) != "" {
			err = fmt.Errorf(errText)
		}
	}
	if err != nil {
		fmt.Printf("redis stream endpoint processing failed: stream=%s message_id=%s ruleChainId=%s error=%v\n", stream, redisMessage.ID, n.config.RuleChainID, err)
		if n.config.AckOnFailure {
			_, _ = n.client.XAck(ctx, stream, n.config.Group, redisMessage.ID).Result()
		}
		return
	}
	_, _ = n.client.XAck(ctx, stream, n.config.Group, redisMessage.ID).Result()
}

func (n *RedisStreamEndpointNode) buildRuleMsg(stream string, redisMessage redis.XMessage) (types.RuleMsg, error) {
	values := map[string]any{
		"redis_stream":     stream,
		"redis_message_id": redisMessage.ID,
	}
	for key, value := range redisMessage.Values {
		values[key] = value
	}
	raw, _ := json.Marshal(values)
	msg := types.NewMsg(n.config.RuleChainID, string(raw), types.Metadata{
		"redis_stream":     stream,
		"redis_message_id": redisMessage.ID,
		"redis_group":      n.config.Group,
	}, types.NewDataT()).WithDataFormat(cnst.JSON)
	if n.config.Input.MapAll != nil || len(n.config.Input.Fields) > 0 {
		nodeCtx := registry.NewMinimalNodeCtx(n.ID())
		if err := helper.ProcessInbound(nodeCtx, msg, n.config.Input, helper.MapProvider(values)); err != nil {
			return nil, err
		}
	}
	return msg, nil
}

func (n *RedisStreamEndpointNode) ensureGroup(ctx context.Context) error {
	err := n.client.XGroupCreateMkStream(ctx, n.config.Stream, n.config.Group, n.config.GroupStartID).Err()
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToUpper(err.Error()), "BUSYGROUP") {
		return nil
	}
	return err
}

func (n *RedisStreamEndpointNode) resolveRuntime() (types.Runtime, bool) {
	if n.runtimePool != nil {
		if rt, ok := n.runtimePool.Get(n.config.RuleChainID); ok && rt != nil {
			return rt, true
		}
	}
	return registry.Default.RuntimePool.Get(n.config.RuleChainID)
}

func (n *RedisStreamEndpointNode) resolveClient() (*redis.Client, error) {
	pool := registry.Default.GetSharedNodePool()
	ast := asset.Asset[*redis.Client]{URI: n.config.RedisClient}
	return ast.Resolve(asset.NewAssetContext(asset.WithNodePool(pool)))
}

func (n *RedisStreamEndpointNode) consumerName(workerID int) string {
	base := strings.TrimSpace(n.config.Consumer)
	if base == "" {
		host, _ := os.Hostname()
		base = fmt.Sprintf("%s-%s-%d", n.ID(), host, os.Getpid())
	}
	if n.config.Concurrency <= 1 {
		return base
	}
	return fmt.Sprintf("%s-%d", base, workerID)
}
