package endpoint

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisStreamEndpointRecoveryRequiresSafeTimeoutBoundary(t *testing.T) {
	node := &RedisStreamEndpointNode{}
	err := node.Init(map[string]any{
		"redisClient":         "ref://redis",
		"stream":              "events",
		"group":               "projector",
		"ruleChainId":         "project-event",
		"processingTimeoutMs": int64(30_000),
		"pendingRecovery": map[string]any{
			"enabled":   true,
			"minIdleMs": int64(30_000),
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "minIdleMs must be greater than processingTimeoutMs")
}

func TestRedisStreamEndpointRejectsAckOnFailureWithRecovery(t *testing.T) {
	node := &RedisStreamEndpointNode{}
	err := node.Init(map[string]any{
		"redisClient":  "ref://redis",
		"stream":       "events",
		"group":        "projector",
		"ruleChainId":  "project-event",
		"ackOnFailure": true,
		"pendingRecovery": map[string]any{
			"enabled": true,
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ackOnFailure must be false")
}

func TestRedisStreamEndpointGeneratesDistinctWorkerConsumers(t *testing.T) {
	node := &RedisStreamEndpointNode{
		config: RedisStreamEndpointConfiguration{Concurrency: 2},
	}
	node.SetID("event-consumer")

	first := node.consumerName(0)
	second := node.consumerName(1)

	assert.NotEmpty(t, first)
	assert.NotEqual(t, first, second)
	assert.Contains(t, first, "event-consumer")
}

func TestRedisStreamEndpointFailedDeliveryStaysPendingBeforeRetryLimit(t *testing.T) {
	transport := &fakeRedisStreamTransport{retryCount: 2}
	node := newRedisStreamDeliveryTestNode(transport)
	node.config.PendingRecovery.MaxDeliveries = 3
	node.config.PendingRecovery.DeadLetterStream = "events.dlq"
	node.processMessage = func(context.Context, string, redis.XMessage) error {
		return errors.New("temporary failure")
	}

	err := node.handleDelivery(context.Background(), "events", redis.XMessage{ID: "1-0"}, "consumer-a")

	require.ErrorContains(t, err, "temporary failure")
	assert.Empty(t, transport.ackedIDs)
	assert.Empty(t, transport.added)
}

func TestRedisStreamEndpointMovesPoisonMessageToDLQBeforeAck(t *testing.T) {
	transport := &fakeRedisStreamTransport{retryCount: 3}
	node := newRedisStreamDeliveryTestNode(transport)
	node.config.PendingRecovery.MaxDeliveries = 3
	node.config.PendingRecovery.DeadLetterStream = "events.dlq"
	node.processMessage = func(context.Context, string, redis.XMessage) error {
		return errors.New("permanent failure")
	}

	err := node.handleDelivery(context.Background(), "events", redis.XMessage{
		ID:     "2-0",
		Values: map[string]any{"event_id": "evt-2", "event_type": "role.changed"},
	}, "consumer-a")

	require.ErrorContains(t, err, "permanent failure")
	require.Len(t, transport.added, 1)
	assert.Equal(t, "events.dlq", transport.added[0].stream)
	assert.Equal(t, "2-0", transport.added[0].values["matrix_original_message_id"])
	assert.Equal(t, int64(3), transport.added[0].values["matrix_delivery_count"])
	assert.Equal(t, []string{"2-0"}, transport.ackedIDs)
}

func TestRedisStreamEndpointDoesNotAckWhenDLQWriteFails(t *testing.T) {
	transport := &fakeRedisStreamTransport{retryCount: 3, addErr: errors.New("redis unavailable")}
	node := newRedisStreamDeliveryTestNode(transport)
	node.config.PendingRecovery.MaxDeliveries = 3
	node.config.PendingRecovery.DeadLetterStream = "events.dlq"
	node.processMessage = func(context.Context, string, redis.XMessage) error {
		return errors.New("permanent failure")
	}

	err := node.handleDelivery(context.Background(), "events", redis.XMessage{ID: "3-0"}, "consumer-a")

	require.ErrorContains(t, err, "dead-letter")
	assert.Empty(t, transport.ackedIDs)
}

func TestRedisStreamEndpointProcessingTimeoutKeepsMessagePending(t *testing.T) {
	transport := &fakeRedisStreamTransport{retryCount: 1}
	node := newRedisStreamDeliveryTestNode(transport)
	node.config.ProcessingTimeoutMs = 10
	node.processMessage = func(ctx context.Context, _ string, _ redis.XMessage) error {
		<-ctx.Done()
		return ctx.Err()
	}

	err := node.handleDelivery(context.Background(), "events", redis.XMessage{ID: "4-0"}, "consumer-a")

	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Empty(t, transport.ackedIDs)
}

func TestRedisStreamEndpointRecoversClaimedMessagesWithSameHandler(t *testing.T) {
	transport := &fakeRedisStreamTransport{
		claimed:    []redis.XMessage{{ID: "5-0", Values: map[string]any{"event_id": "evt-5"}}},
		nextCursor: "7-0",
	}
	node := newRedisStreamDeliveryTestNode(transport)
	var processed []string
	node.processMessage = func(_ context.Context, _ string, message redis.XMessage) error {
		processed = append(processed, message.ID)
		return nil
	}

	next, err := node.recoverPendingBatch(context.Background(), "consumer-a", "0-0")

	require.NoError(t, err)
	assert.Equal(t, "7-0", next)
	assert.Equal(t, []string{"5-0"}, processed)
	assert.Equal(t, []string{"5-0"}, transport.ackedIDs)
	assert.Equal(t, "consumer-a", transport.claimConsumer)
}

func newRedisStreamDeliveryTestNode(transport redisStreamTransport) *RedisStreamEndpointNode {
	return &RedisStreamEndpointNode{
		config: RedisStreamEndpointConfiguration{
			Stream: "events",
			Group:  "projector",
			Count:  10,
			PendingRecovery: RedisStreamPendingRecoveryConfiguration{
				Enabled:   true,
				MinIdleMs: 60_000,
				Count:     10,
			},
		},
		transport: transport,
	}
}

type fakeRedisStreamTransport struct {
	mu            sync.Mutex
	retryCount    int64
	claimed       []redis.XMessage
	nextCursor    string
	claimConsumer string
	ackedIDs      []string
	added         []fakeRedisStreamAdd
	addErr        error
}

type fakeRedisStreamAdd struct {
	stream string
	values map[string]any
}

func (f *fakeRedisStreamTransport) readNew(context.Context, string, string, string, int64, time.Duration) ([]redis.XStream, error) {
	return nil, redis.Nil
}

func (f *fakeRedisStreamTransport) autoClaim(_ context.Context, _ string, _ string, consumer string, _ time.Duration, _ string, _ int64) ([]redis.XMessage, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.claimConsumer = consumer
	return append([]redis.XMessage(nil), f.claimed...), f.nextCursor, nil
}

func (f *fakeRedisStreamTransport) pendingRetryCount(context.Context, string, string, string) (int64, error) {
	return f.retryCount, nil
}

func (f *fakeRedisStreamTransport) ack(_ context.Context, _ string, _ string, ids ...string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ackedIDs = append(f.ackedIDs, ids...)
	return nil
}

func (f *fakeRedisStreamTransport) add(_ context.Context, stream string, values map[string]any) error {
	if f.addErr != nil {
		return f.addErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	f.added = append(f.added, fakeRedisStreamAdd{stream: stream, values: cloned})
	return nil
}

func (f *fakeRedisStreamTransport) createGroup(context.Context, string, string, string) error {
	return nil
}
