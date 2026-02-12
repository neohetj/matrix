package external

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/neohetj/matrix/internal/builtin/base"
	"github.com/neohetj/matrix/pkg/asset"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/pkg/utils"
	"github.com/redis/go-redis/v9"
)

const (
	RedisClientNodeType = "external/redisClient"
)

var (
	RedisParseDSNFailed = &types.Fault{Code: cnst.CodeRedisParseDSNFailed, Message: "failed to parse redis URI"}
	RedisConnectFailed  = &types.Fault{Code: cnst.CodeRedisConnectFailed, Message: "failed to connect to redis"}
)

// RedisClientNodePrototype is the shared prototype instance for registration.
var RedisClientNodePrototype = &RedisClientNode{
	BaseNode: *types.NewBaseNode(RedisClientNodeType, types.NodeMetadata{
		Name:        "Redis Client",
		Description: "Provides a shared redis connection client (*redis.Client).",
		Dimension:   "External",
		Tags:        []string{"external", "database", "redis", "nosql"},
		Version:     "1.0.0",
	}),
}

func init() {
	types.DefaultRegistry.GetNodeManager().Register(RedisClientNodePrototype)
	types.DefaultRegistry.GetFaultRegistry().Register(RedisConnectFailed, RedisParseDSNFailed)
}

// RedisClientNodeConfiguration holds the configuration for the RedisClientNode.
type RedisClientNodeConfiguration struct {
	URI         string `json:"uri"`
	PoolSize    int    `json:"poolSize"`
	TLSInsecure bool   `json:"tls_insecure"`
}

// RedisClientNode is a component that provides a shared redis connection client (*redis.Client).
type RedisClientNode struct {
	types.BaseNode
	types.Instance
	base.Shareable[*redis.Client]
	nodeConfig RedisClientNodeConfiguration
	client     *redis.Client
	closeOnce  sync.Once
}

// New creates a new instance of the RedisClientNode.
func (n *RedisClientNode) New() types.Node {
	return &RedisClientNode{
		BaseNode: n.BaseNode,
	}
}

// Init initializes the node.
func (n *RedisClientNode) Init(cfg types.ConfigMap) error {
	if err := utils.Decode(cfg, &n.nodeConfig); err != nil {
		return fmt.Errorf("failed to decode redis client node config: %w", err)
	}

	uri, err := asset.RenderTemplate(n.nodeConfig.URI, asset.NewAssetContext())
	if err != nil {
		return fmt.Errorf("failed to render uri template: %s, error: %w", n.nodeConfig.URI, err)
	}
	n.nodeConfig.URI = uri

	initFunc := func() (*redis.Client, error) {
		if n.client != nil {
			return n.client, nil
		}

		opt, err := redis.ParseURL(n.nodeConfig.URI)
		if err != nil {
			return nil, RedisParseDSNFailed.Wrap(err)
		}

		if n.nodeConfig.PoolSize > 0 {
			opt.PoolSize = n.nodeConfig.PoolSize
		}

		if n.nodeConfig.TLSInsecure {
			if opt.TLSConfig == nil {
				opt.TLSConfig = &tls.Config{}
			}
			opt.TLSConfig.InsecureSkipVerify = true
		}

		client := redis.NewClient(opt)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := client.Ping(ctx).Err(); err != nil {
			return nil, RedisConnectFailed.Wrap(err)
		}

		n.client = client
		return n.client, nil
	}

	return n.Shareable.Init(nil, n.nodeConfig.URI, initFunc)
}

// OnMsg for a resource node is typically a no-op.
func (n *RedisClientNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {
	// No-op
}

// Errors returns the list of possible faults that this node can produce.
func (n *RedisClientNode) Errors() []*types.Fault {
	return append(n.Shareable.Errors(), RedisConnectFailed)
}

// Destroy closes the redis connection if it was created by this node.
func (n *RedisClientNode) Destroy() {
	n.closeOnce.Do(func() {
		if n.client != nil {
			if err := n.client.Close(); err != nil {
				// Log error?
			}
		}
	})
}
