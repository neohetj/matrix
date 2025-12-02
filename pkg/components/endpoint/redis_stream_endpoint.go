package endpoint

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gitlab.com/neohet/matrix/internal/log"
	redisutils "gitlab.com/neohet/matrix/pkg/connectors/redis"
	"gitlab.com/neohet/matrix/pkg/registry"
	"gitlab.com/neohet/matrix/pkg/types"
	"gitlab.com/neohet/matrix/pkg/utils"
)

const (
	RedisStreamEndpointType = "endpoint/redisStream"
	TraceIDKey              = "trace_id"
)

var redisStreamEndpointPrototype = &RedisStreamEndpointNode{
	BaseNode: *types.NewBaseNode(RedisStreamEndpointType, types.NodeDefinition{
		Name:        "Redis Stream Endpoint",
		Description: "Actively consumes messages from a Redis Stream.",
		Dimension:   "Endpoint",
		Tags:        []string{"endpoint", "redis", "stream"},
		Version:     "1.0.0",
	}),
}

func init() {
	registry.Default.NodeManager.Register(redisStreamEndpointPrototype)
}

// RedisStreamEndpointNodeConfiguration defines the configuration for the RedisStreamEndpointNode.
type RedisStreamEndpointNodeConfiguration struct {
	// RuleChainID is the ID of the rule chain to be executed when a message is received.
	RuleChainID string `json:"ruleChainId"`
	// StartNodeID is an optional ID of a specific node within the rule chain to start execution from.
	// If empty, execution starts from the root node(s).
	StartNodeID string `json:"startNodeId,omitempty"`
	// RedisDsn is the DSN or reference (e.g., "ref://shared_redis") to the Redis instance.
	RedisDsn string `json:"redisDsn"`
	// StreamKey is the name of the Redis Stream to consume from.
	StreamKey string `json:"streamKey"`
	// ConsumerGroup is the name of the consumer group.
	ConsumerGroup string `json:"consumerGroup"`
	// ConsumerName is the unique name for this consumer within the group.
	ConsumerName string `json:"consumerName"`
	// BatchSize is the maximum number of messages to fetch in a single XReadGroup call.
	BatchSize int64 `json:"batchSize"`
	// BlockTimeout is the maximum time to wait for a message before the XReadGroup call returns.
	BlockTimeout string `json:"blockTimeout"`
	// Description provides a human-readable description of the endpoint's purpose.
	Description string `json:"description"`
}

// RedisStreamEndpointNode is an active endpoint that consumes messages from a Redis Stream.
type RedisStreamEndpointNode struct {
	types.BaseNode
	types.Instance
	nodeConfig   RedisStreamEndpointNodeConfiguration
	redisClient  *redis.Client
	runtimePool  types.RuntimePool
	logger       types.Logger
	cancel       context.CancelFunc
	blockTimeout time.Duration
}

// New creates a new instance of the node.
func (n *RedisStreamEndpointNode) New() types.Node {
	return &RedisStreamEndpointNode{
		BaseNode: n.BaseNode,
		logger:   log.GetLogger(),
	}
}

// Init initializes the node with its configuration.
func (n *RedisStreamEndpointNode) Init(config types.Config) error {
	err := utils.Decode(config, &n.nodeConfig)
	if err != nil {
		return fmt.Errorf("failed to decode node configuration: %w", err)
	}

	if n.nodeConfig.RuleChainID == "" {
		return fmt.Errorf("ruleChainId is required")
	}
	if n.nodeConfig.RedisDsn == "" {
		return fmt.Errorf("RedisDsn is required")
	}
	if n.nodeConfig.StreamKey == "" {
		return fmt.Errorf("streamKey is required")
	}
	if n.nodeConfig.ConsumerGroup == "" {
		return fmt.Errorf("consumerGroup is required")
	}
	if n.nodeConfig.ConsumerName == "" {
		return fmt.Errorf("consumerName is required")
	}
	if n.nodeConfig.BatchSize <= 0 {
		n.nodeConfig.BatchSize = 1
	}
	if n.nodeConfig.BlockTimeout == "" {
		n.nodeConfig.BlockTimeout = "5s"
	}

	n.blockTimeout, err = time.ParseDuration(n.nodeConfig.BlockTimeout)
	if err != nil {
		return fmt.Errorf("invalid blockTimeout duration: %w", err)
	}

	return nil
}

// SetRuntimePool injects the runtime pool.
func (n *RedisStreamEndpointNode) SetRuntimePool(pool interface{}) error {
	if p, ok := pool.(types.RuntimePool); ok {
		n.runtimePool = p
		return nil
	}
	return fmt.Errorf("provided pool is not of type types.RuntimePool")
}

// GetInstance implements the types.SharedNode interface.
func (n *RedisStreamEndpointNode) GetInstance() (interface{}, error) {
	return n, nil
}

// GetConfiguration returns the node's configuration for inspection.
func (n *RedisStreamEndpointNode) Configuration() RedisStreamEndpointNodeConfiguration {
	return n.nodeConfig
}

// Start implements the ActiveEndpoint interface. It initializes the Redis connection,
// ensures the consumer group exists, and starts the background consumption loop.
func (n *RedisStreamEndpointNode) Start(ctx context.Context) error {
	// Get the specific runtime instance for the configured rule chain.
	rt, ok := n.runtimePool.Get(n.nodeConfig.RuleChainID)
	if !ok {
		return fmt.Errorf("runtime for rule chain '%s' not found in pool", n.nodeConfig.RuleChainID)
	}

	// Get the NodePool from the runtime instance.
	nodePool := rt.GetNodePool()
	if nodePool == nil {
		return fmt.Errorf("nodePool is not available in runtime for rule chain '%s'", n.nodeConfig.RuleChainID)
	}

	client, isTemp, err := redisutils.GetRedisConnection(nodePool, n.nodeConfig.RedisDsn)
	if err != nil {
		return fmt.Errorf("failed to get redis connection for ref %s: %w", n.nodeConfig.RedisDsn, err)
	}
	if isTemp {
		// Endpoints should always use shared, non-temporary connections.
		client.Close()
		return fmt.Errorf("redis connection for an endpoint must be a shared reference (ref://), not a DSN")
	}
	n.redisClient = client

	// Ensure the consumer group exists.
	err = n.redisClient.XGroupCreateMkStream(ctx, n.nodeConfig.StreamKey, n.nodeConfig.ConsumerGroup, "$").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	ctx, n.cancel = context.WithCancel(ctx)

	go n.consumeLoop(ctx)

	n.logger.Infof(ctx, "Redis Stream consumer started for stream '%s' with group '%s'", n.nodeConfig.StreamKey, n.nodeConfig.ConsumerGroup)
	return nil
}

// Stop implements the ActiveEndpoint interface. It gracefully stops the consumer loop
// by canceling its context.
func (n *RedisStreamEndpointNode) Stop() error {
	if n.cancel != nil {
		n.cancel()
	}
	return nil
}

// Destroy cleans up resources used by the node.
func (n *RedisStreamEndpointNode) Destroy() {
	n.Stop()
}

// consumeLoop is the main loop that continuously polls the Redis Stream for new messages.
func (n *RedisStreamEndpointNode) consumeLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			n.logger.Infof(ctx, "Stopping Redis Stream consumer for stream '%s'", n.nodeConfig.StreamKey)
			return
		default:
			streams, err := n.redisClient.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    n.nodeConfig.ConsumerGroup,
				Consumer: n.nodeConfig.ConsumerName,
				Streams:  []string{n.nodeConfig.StreamKey, ">"},
				Count:    n.nodeConfig.BatchSize,
				Block:    n.blockTimeout,
				NoAck:    false,
			}).Result()

			if err != nil {
				if err == redis.Nil || err == context.Canceled {
					continue
				}
				n.logger.Errorf(ctx, "Error reading from stream '%s': %v", n.nodeConfig.StreamKey, err)
				time.Sleep(time.Second) // Wait a bit before retrying on error.
				continue
			}

			for _, stream := range streams {
				for _, message := range stream.Messages {
					n.handleMessage(ctx, message)
				}
			}
		}
	}
}

// handleMessage processes a single message received from the Redis Stream.
// It constructs a new RuleMsg, sets the message content as its data, and triggers the rule chain execution.
func (n *RedisStreamEndpointNode) handleMessage(ctx context.Context, message redis.XMessage) {
	n.logger.Infof(ctx, "Handling message from Redis Stream.", "stream", n.nodeConfig.StreamKey, "messageId", message.ID, "values", message.Values)

	// The message values (a map) are marshaled into a JSON string to be placed in the RuleMsg's data field.
	// Downstream nodes will need to unmarshal this JSON to access the data.
	msgBody, err := json.Marshal(message.Values)
	if err != nil {
		n.logger.Errorf(ctx, "Failed to marshal message values to JSON for message ID %s: %v", message.ID, err)
		return
	}

	metadata := make(types.Metadata)
	metadata["stream"] = n.nodeConfig.StreamKey
	metadata["group"] = n.nodeConfig.ConsumerGroup
	metadata["consumer"] = n.nodeConfig.ConsumerName
	metadata["messageId"] = message.ID

	// Attempt to extract propagated metadata, including trace context.
	if metaJSON, ok := message.Values["__matrix_metadata"].(string); ok {
		var propagatedMeta map[string]string
		if err := json.Unmarshal([]byte(metaJSON), &propagatedMeta); err == nil {
			for k, v := range propagatedMeta {
				metadata[k] = v
			}
			n.logger.Debugf(ctx, "Successfully extracted propagated metadata for message %s.", message.ID)
		} else {
			n.logger.Warnf(ctx, "Failed to unmarshal propagated metadata for message %s: %v", message.ID, err)
		}
	} else {
		// Fallback for older or non-compliant producers.
		if traceID, ok := message.Values[TraceIDKey].(string); ok {
			metadata[TraceIDKey] = traceID
		}
	}

	msg := types.NewMsg(n.nodeConfig.RuleChainID, "", metadata, nil).WithDataFormat(types.JSON)
	msg.SetData(string(msgBody))

	rt, ok := n.runtimePool.Get(n.nodeConfig.RuleChainID)
	if !ok {
		n.logger.Errorf(ctx, "Rule chain '%s' not found in runtime pool", n.nodeConfig.RuleChainID)
		return
	}

	onEnd := func(msg types.RuleMsg, err error) {
		if err != nil {
			n.logger.Errorf(ctx, "Rule chain execution failed for message %s: %v", message.ID, err)
		} else {
			n.logger.Debugf(ctx, "Rule chain execution completed successfully for message %s.", message.ID)
		}
	}

	// Asynchronously execute the rule chain.
	if err := rt.Execute(ctx, n.nodeConfig.StartNodeID, msg, onEnd); err != nil {
		n.logger.Errorf(ctx, "Failed to execute rule chain for message %s: %v", message.ID, err)
	} else {
		n.logger.Debugf(ctx, "Successfully dispatched message %s to rule chain %s.", message.ID, n.nodeConfig.RuleChainID)
	}
}
