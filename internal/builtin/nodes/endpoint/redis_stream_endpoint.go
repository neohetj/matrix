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
		Version:     "1.1.0",
		Icon:        "message-queue",
	}),
}

func init() {
	registry.Default.GetNodeManager().Register(redisStreamEndpointNodePrototype)
	registry.Default.GetFaultRegistry().Register(FaultRedisStreamEndpointConfig, FaultRedisStreamEndpointRuntime)
}

type RedisStreamEndpointConfiguration struct {
	RedisClient         string                                  `json:"redisClient"`
	Stream              string                                  `json:"stream"`
	Group               string                                  `json:"group"`
	Consumer            string                                  `json:"consumer,omitempty"`
	RuleChainID         string                                  `json:"ruleChainId"`
	StartNodeID         string                                  `json:"startNodeId,omitempty"`
	Input               types.EndpointIOPacket                  `json:"input,omitempty"`
	Count               int64                                   `json:"count,omitempty"`
	BlockMs             int64                                   `json:"blockMs,omitempty"`
	Concurrency         int                                     `json:"concurrency,omitempty"`
	AutoCreateGroup     bool                                    `json:"autoCreateGroup,omitempty"`
	GroupStartID        string                                  `json:"groupStartId,omitempty"`
	AckOnFailure        bool                                    `json:"ackOnFailure,omitempty"`
	ProcessingTimeoutMs int64                                   `json:"processingTimeoutMs,omitempty"`
	PendingRecovery     RedisStreamPendingRecoveryConfiguration `json:"pendingRecovery,omitempty"`
}

type RedisStreamPendingRecoveryConfiguration struct {
	Enabled          bool   `json:"enabled,omitempty"`
	MinIdleMs        int64  `json:"minIdleMs,omitempty"`
	IntervalMs       int64  `json:"intervalMs,omitempty"`
	Count            int64  `json:"count,omitempty"`
	MaxDeliveries    int64  `json:"maxDeliveries,omitempty"`
	DeadLetterStream string `json:"deadLetterStream,omitempty"`
}

type RedisStreamEndpointNode struct {
	types.BaseNode
	types.Instance
	config         RedisStreamEndpointConfiguration
	runtimePool    types.RuntimePool
	transport      redisStreamTransport
	processMessage func(context.Context, string, redis.XMessage) error
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	startOnce      sync.Once
	stopOnce       sync.Once
	startErr       error
}

type redisStreamTransport interface {
	readNew(ctx context.Context, stream, group, consumer string, count int64, block time.Duration) ([]redis.XStream, error)
	autoClaim(ctx context.Context, stream, group, consumer string, minIdle time.Duration, start string, count int64) ([]redis.XMessage, string, error)
	pendingRetryCount(ctx context.Context, stream, group, messageID string) (int64, error)
	ack(ctx context.Context, stream, group string, ids ...string) error
	add(ctx context.Context, stream string, values map[string]any) error
	createGroup(ctx context.Context, stream, group, startID string) error
}

type goRedisStreamTransport struct {
	client *redis.Client
}

func (t *goRedisStreamTransport) readNew(ctx context.Context, stream, group, consumer string, count int64, block time.Duration) ([]redis.XStream, error) {
	return t.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{stream, ">"},
		Count:    count,
		Block:    block,
	}).Result()
}

func (t *goRedisStreamTransport) autoClaim(ctx context.Context, stream, group, consumer string, minIdle time.Duration, start string, count int64) ([]redis.XMessage, string, error) {
	return t.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   stream,
		Group:    group,
		Consumer: consumer,
		MinIdle:  minIdle,
		Start:    start,
		Count:    count,
	}).Result()
}

func (t *goRedisStreamTransport) pendingRetryCount(ctx context.Context, stream, group, messageID string) (int64, error) {
	entries, err := t.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: stream,
		Group:  group,
		Start:  messageID,
		End:    messageID,
		Count:  1,
	}).Result()
	if err != nil {
		return 0, err
	}
	if len(entries) == 0 || entries[0].ID != messageID {
		return 0, fmt.Errorf("pending message %s not found", messageID)
	}
	return entries[0].RetryCount, nil
}

func (t *goRedisStreamTransport) ack(ctx context.Context, stream, group string, ids ...string) error {
	return t.client.XAck(ctx, stream, group, ids...).Err()
}

func (t *goRedisStreamTransport) add(ctx context.Context, stream string, values map[string]any) error {
	return t.client.XAdd(ctx, &redis.XAddArgs{Stream: stream, Values: values}).Err()
}

func (t *goRedisStreamTransport) createGroup(ctx context.Context, stream, group, startID string) error {
	return t.client.XGroupCreateMkStream(ctx, stream, group, startID).Err()
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
	if n.config.ProcessingTimeoutMs < 0 {
		return FaultRedisStreamEndpointConfig.Wrap(fmt.Errorf("processingTimeoutMs must not be negative"))
	}
	if n.config.PendingRecovery.Enabled {
		if n.config.AckOnFailure {
			return FaultRedisStreamEndpointConfig.Wrap(fmt.Errorf("ackOnFailure must be false when pendingRecovery is enabled"))
		}
		if n.config.ProcessingTimeoutMs == 0 {
			n.config.ProcessingTimeoutMs = 30000
		}
		if n.config.PendingRecovery.MinIdleMs <= 0 {
			n.config.PendingRecovery.MinIdleMs = max(60000, n.config.ProcessingTimeoutMs+30000)
		}
		if n.config.PendingRecovery.MinIdleMs <= n.config.ProcessingTimeoutMs {
			return FaultRedisStreamEndpointConfig.Wrap(fmt.Errorf("pendingRecovery.minIdleMs must be greater than processingTimeoutMs"))
		}
		if n.config.PendingRecovery.IntervalMs <= 0 {
			n.config.PendingRecovery.IntervalMs = 10000
		}
		if n.config.PendingRecovery.Count <= 0 {
			n.config.PendingRecovery.Count = n.config.Count
		}
		if n.config.PendingRecovery.MaxDeliveries < 0 {
			return FaultRedisStreamEndpointConfig.Wrap(fmt.Errorf("pendingRecovery.maxDeliveries must not be negative"))
		}
		if n.config.PendingRecovery.MaxDeliveries > 0 && strings.TrimSpace(n.config.PendingRecovery.DeadLetterStream) == "" {
			return FaultRedisStreamEndpointConfig.Wrap(fmt.Errorf("pendingRecovery.deadLetterStream is required when maxDeliveries is set"))
		}
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
		startedAt := time.Now()
		fmt.Printf("redis stream endpoint starting: node_id=%s stream=%s group=%s consumer=%s ruleChainId=%s concurrency=%d blockMs=%d autoCreateGroup=%t pendingRecovery=%t processingTimeoutMs=%d minIdleMs=%d maxDeliveries=%d deadLetterStream=%s\n",
			n.ID(), n.config.Stream, n.config.Group, n.config.Consumer, n.config.RuleChainID, n.config.Concurrency, n.config.BlockMs, n.config.AutoCreateGroup,
			n.config.PendingRecovery.Enabled, n.config.ProcessingTimeoutMs, n.config.PendingRecovery.MinIdleMs, n.config.PendingRecovery.MaxDeliveries, n.config.PendingRecovery.DeadLetterStream)
		client, err := n.resolveClient()
		if err != nil {
			fmt.Printf("redis stream endpoint resolve client failed: node_id=%s stream=%s group=%s redisClient=%s duration_ms=%d error=%v\n",
				n.ID(), n.config.Stream, n.config.Group, n.config.RedisClient, endpointElapsedMilliseconds(startedAt), err)
			n.startErr = err
			return
		}
		n.transport = &goRedisStreamTransport{client: client}
		n.ctx, n.cancel = context.WithCancel(ctx)
		if n.config.AutoCreateGroup {
			if err := n.ensureGroup(n.ctx); err != nil {
				fmt.Printf("redis stream endpoint ensure group failed: node_id=%s stream=%s group=%s groupStartId=%s duration_ms=%d error=%v\n",
					n.ID(), n.config.Stream, n.config.Group, n.config.GroupStartID, endpointElapsedMilliseconds(startedAt), err)
				n.startErr = err
				return
			}
			fmt.Printf("redis stream endpoint group ready: node_id=%s stream=%s group=%s groupStartId=%s duration_ms=%d\n",
				n.ID(), n.config.Stream, n.config.Group, n.config.GroupStartID, endpointElapsedMilliseconds(startedAt))
		}
		for i := 0; i < n.config.Concurrency; i++ {
			n.wg.Add(1)
			go n.runWorker(i)
		}
		fmt.Printf("redis stream endpoint started: node_id=%s stream=%s group=%s concurrency=%d duration_ms=%d\n",
			n.ID(), n.config.Stream, n.config.Group, n.config.Concurrency, endpointElapsedMilliseconds(startedAt))
	})
	return n.startErr
}

func (n *RedisStreamEndpointNode) Stop() error {
	n.stopOnce.Do(func() {
		fmt.Printf("redis stream endpoint stopping: node_id=%s stream=%s group=%s\n", n.ID(), n.config.Stream, n.config.Group)
		if n.cancel != nil {
			n.cancel()
		}
		n.wg.Wait()
		fmt.Printf("redis stream endpoint stopped: node_id=%s stream=%s group=%s\n", n.ID(), n.config.Stream, n.config.Group)
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
	consumer := n.consumerName(workerID)
	recoveryCursor := "0-0"
	nextRecoveryAt := time.Now().Add(n.recoveryInitialDelay(consumer))
	fmt.Printf("redis stream endpoint worker started: node_id=%s stream=%s group=%s consumer=%s worker_id=%d\n",
		n.ID(), n.config.Stream, n.config.Group, consumer, workerID)
	defer func() {
		fmt.Printf("redis stream endpoint worker stopped: node_id=%s stream=%s group=%s consumer=%s worker_id=%d\n",
			n.ID(), n.config.Stream, n.config.Group, consumer, workerID)
		n.wg.Done()
	}()
	for {
		select {
		case <-n.ctx.Done():
			return
		default:
		}

		if n.config.PendingRecovery.Enabled && !time.Now().Before(nextRecoveryAt) {
			nextCursor, err := n.recoverPendingBatch(n.ctx, consumer, recoveryCursor)
			if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				fmt.Printf("redis stream endpoint pending recovery failed: node_id=%s stream=%s group=%s consumer=%s cursor=%s error=%v\n",
					n.ID(), n.config.Stream, n.config.Group, consumer, recoveryCursor, err)
			}
			if strings.TrimSpace(nextCursor) != "" {
				recoveryCursor = nextCursor
			}
			nextRecoveryAt = time.Now().Add(time.Duration(n.config.PendingRecovery.IntervalMs) * time.Millisecond)
		}

		streams, err := n.transport.readNew(
			n.ctx,
			n.config.Stream,
			n.config.Group,
			consumer,
			n.config.Count,
			time.Duration(n.config.BlockMs)*time.Millisecond,
		)
		if err != nil {
			if errors.Is(err, redis.Nil) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				continue
			}
			fmt.Printf("redis stream endpoint read error: stream=%s group=%s consumer=%s error=%v\n", n.config.Stream, n.config.Group, consumer, err)
			if !waitRedisStreamEndpoint(n.ctx, time.Second) {
				return
			}
			continue
		}
		messageCount := 0
		for _, stream := range streams {
			messageCount += len(stream.Messages)
		}
		if messageCount > 0 {
			fmt.Printf("redis stream endpoint read messages: node_id=%s stream=%s group=%s consumer=%s message_count=%d\n",
				n.ID(), n.config.Stream, n.config.Group, consumer, messageCount)
		}
		for _, stream := range streams {
			for _, redisMessage := range stream.Messages {
				_ = n.handleDelivery(n.ctx, stream.Stream, redisMessage, consumer)
			}
		}
	}
}

func (n *RedisStreamEndpointNode) handleDelivery(ctx context.Context, stream string, redisMessage redis.XMessage, consumer string) error {
	startedAt := time.Now()
	fmt.Printf("redis stream endpoint message received: node_id=%s stream=%s group=%s message_id=%s event_id=%s event_type=%s tenant_id=%s idempotency_key=%s\n",
		n.ID(), stream, n.config.Group, redisMessage.ID,
		redisEndpointMessageValue(redisMessage, "event_id"),
		redisEndpointMessageValue(redisMessage, "event_type"),
		redisEndpointMessageValue(redisMessage, "tenant_id"),
		redisEndpointMessageValue(redisMessage, "idempotency_key"),
	)
	processingCtx := ctx
	cancel := func() {}
	if n.config.ProcessingTimeoutMs > 0 {
		processingCtx, cancel = context.WithTimeout(ctx, time.Duration(n.config.ProcessingTimeoutMs)*time.Millisecond)
	}
	defer cancel()
	processor := n.processMessage
	if processor == nil {
		processor = n.executeMessage
	}
	err := processor(processingCtx, stream, redisMessage)
	if err != nil {
		fmt.Printf("redis stream endpoint processing failed: node_id=%s stream=%s group=%s message_id=%s event_id=%s event_type=%s tenant_id=%s idempotency_key=%s ruleChainId=%s duration_ms=%d error=%v\n",
			n.ID(), stream, n.config.Group, redisMessage.ID,
			redisEndpointMessageValue(redisMessage, "event_id"),
			redisEndpointMessageValue(redisMessage, "event_type"),
			redisEndpointMessageValue(redisMessage, "tenant_id"),
			redisEndpointMessageValue(redisMessage, "idempotency_key"),
			n.config.RuleChainID,
			endpointElapsedMilliseconds(startedAt),
			err)
		if n.config.AckOnFailure {
			if ackErr := n.transport.ack(ctx, stream, n.config.Group, redisMessage.ID); ackErr != nil {
				fmt.Printf("redis stream endpoint ack failed after processing failure: node_id=%s stream=%s group=%s message_id=%s error=%v\n",
					n.ID(), stream, n.config.Group, redisMessage.ID, ackErr)
				return errors.Join(err, ackErr)
			}
			return err
		}
		if dlqErr := n.deadLetterIfExhausted(ctx, stream, redisMessage, consumer, err); dlqErr != nil {
			return errors.Join(err, dlqErr)
		}
		return err
	}
	if err := n.transport.ack(ctx, stream, n.config.Group, redisMessage.ID); err != nil {
		fmt.Printf("redis stream endpoint ack failed: node_id=%s stream=%s group=%s message_id=%s duration_ms=%d error=%v\n",
			n.ID(), stream, n.config.Group, redisMessage.ID, endpointElapsedMilliseconds(startedAt), err)
		return err
	}
	fmt.Printf("redis stream endpoint message processed: node_id=%s stream=%s group=%s message_id=%s event_id=%s event_type=%s tenant_id=%s idempotency_key=%s ruleChainId=%s duration_ms=%d\n",
		n.ID(), stream, n.config.Group, redisMessage.ID,
		redisEndpointMessageValue(redisMessage, "event_id"),
		redisEndpointMessageValue(redisMessage, "event_type"),
		redisEndpointMessageValue(redisMessage, "tenant_id"),
		redisEndpointMessageValue(redisMessage, "idempotency_key"),
		n.config.RuleChainID,
		endpointElapsedMilliseconds(startedAt),
	)
	return nil
}

func (n *RedisStreamEndpointNode) executeMessage(ctx context.Context, stream string, redisMessage redis.XMessage) error {
	rt, ok := n.resolveRuntime()
	if !ok {
		return fmt.Errorf("rule chain runtime %s not found", n.config.RuleChainID)
	}
	msg, err := n.buildRuleMsg(stream, redisMessage)
	if err != nil {
		return fmt.Errorf("build rule message: %w", err)
	}
	finalMsg, err := rt.ExecuteAndWait(ctx, n.config.StartNodeID, msg, nil)
	if err == nil && finalMsg != nil {
		if errText, failed := finalMsg.Metadata()[types.MetaError]; failed && strings.TrimSpace(errText) != "" {
			err = errors.New(errText)
		}
	}
	return err
}

func (n *RedisStreamEndpointNode) recoverPendingBatch(ctx context.Context, consumer, cursor string) (string, error) {
	messages, next, err := n.transport.autoClaim(
		ctx,
		n.config.Stream,
		n.config.Group,
		consumer,
		time.Duration(n.config.PendingRecovery.MinIdleMs)*time.Millisecond,
		cursor,
		n.config.PendingRecovery.Count,
	)
	if err != nil {
		return cursor, err
	}
	if len(messages) > 0 {
		fmt.Printf("redis stream endpoint pending messages claimed: node_id=%s stream=%s group=%s consumer=%s message_count=%d cursor=%s next_cursor=%s\n",
			n.ID(), n.config.Stream, n.config.Group, consumer, len(messages), cursor, next)
	}
	for _, message := range messages {
		_ = n.handleDelivery(ctx, n.config.Stream, message, consumer)
	}
	if strings.TrimSpace(next) == "" {
		return "0-0", nil
	}
	return next, nil
}

func (n *RedisStreamEndpointNode) deadLetterIfExhausted(ctx context.Context, stream string, message redis.XMessage, consumer string, processingErr error) error {
	recovery := n.config.PendingRecovery
	if !recovery.Enabled || recovery.MaxDeliveries <= 0 {
		return nil
	}
	deliveryCount, err := n.transport.pendingRetryCount(ctx, stream, n.config.Group, message.ID)
	if err != nil {
		return fmt.Errorf("read pending delivery count: %w", err)
	}
	if deliveryCount < recovery.MaxDeliveries {
		return nil
	}
	values := make(map[string]any, len(message.Values)+8)
	for key, value := range message.Values {
		values[key] = value
	}
	values["matrix_original_stream"] = stream
	values["matrix_original_group"] = n.config.Group
	values["matrix_original_consumer"] = consumer
	values["matrix_original_message_id"] = message.ID
	values["matrix_delivery_count"] = deliveryCount
	values["matrix_failed_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	values["matrix_failure"] = processingErr.Error()
	if err := n.transport.add(ctx, recovery.DeadLetterStream, values); err != nil {
		return fmt.Errorf("dead-letter write failed: %w", err)
	}
	if err := n.transport.ack(ctx, stream, n.config.Group, message.ID); err != nil {
		return fmt.Errorf("dead-letter ack failed: %w", err)
	}
	fmt.Printf("redis stream endpoint message dead-lettered: node_id=%s stream=%s group=%s message_id=%s consumer=%s delivery_count=%d dead_letter_stream=%s\n",
		n.ID(), stream, n.config.Group, message.ID, consumer, deliveryCount, recovery.DeadLetterStream)
	return nil
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
	err := n.transport.createGroup(ctx, n.config.Stream, n.config.Group, n.config.GroupStartID)
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

func (n *RedisStreamEndpointNode) recoveryInitialDelay(consumer string) time.Duration {
	if !n.config.PendingRecovery.Enabled || n.config.PendingRecovery.IntervalMs <= 0 {
		return 0
	}
	interval := time.Duration(n.config.PendingRecovery.IntervalMs) * time.Millisecond
	var hash uint64
	for index := range consumer {
		hash = hash*33 + uint64(consumer[index])
	}
	return time.Duration(hash % uint64(interval))
}

func waitRedisStreamEndpoint(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func endpointElapsedMilliseconds(startedAt time.Time) int64 {
	return time.Since(startedAt).Milliseconds()
}

func redisEndpointMessageValue(message redis.XMessage, key string) string {
	value, ok := message.Values[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	default:
		return fmt.Sprintf("%v", typed)
	}
}
