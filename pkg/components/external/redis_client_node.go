package external

import (
	"crypto/tls"
	"fmt"
	"strings"
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
	URI             string `json:"uri"`
	PoolSize        int    `json:"poolSize"`
	TLSInsecure     bool   `json:"tls_insecure"`
	DialTimeout     string `json:"dialTimeout"`
	ReadTimeout     string `json:"readTimeout"`
	WriteTimeout    string `json:"writeTimeout"`
	MaxRetries      int    `json:"maxRetries"`
	MinRetryBackoff string `json:"minRetryBackoff"`
	MaxRetryBackoff string `json:"maxRetryBackoff"`
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

	if err := validateRedisDurationOptions(n.nodeConfig); err != nil {
		return err
	}

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

		if err := applyRedisDurationOption("dialTimeout", n.nodeConfig.DialTimeout, func(v time.Duration) {
			opt.DialTimeout = v
		}); err != nil {
			return nil, RedisParseDSNFailed.Wrap(err)
		}
		if err := applyRedisDurationOption("readTimeout", n.nodeConfig.ReadTimeout, func(v time.Duration) {
			opt.ReadTimeout = v
		}); err != nil {
			return nil, RedisParseDSNFailed.Wrap(err)
		}
		if err := applyRedisDurationOption("writeTimeout", n.nodeConfig.WriteTimeout, func(v time.Duration) {
			opt.WriteTimeout = v
		}); err != nil {
			return nil, RedisParseDSNFailed.Wrap(err)
		}
		if err := applyRedisDurationOption("minRetryBackoff", n.nodeConfig.MinRetryBackoff, func(v time.Duration) {
			opt.MinRetryBackoff = v
		}); err != nil {
			return nil, RedisParseDSNFailed.Wrap(err)
		}
		if err := applyRedisDurationOption("maxRetryBackoff", n.nodeConfig.MaxRetryBackoff, func(v time.Duration) {
			opt.MaxRetryBackoff = v
		}); err != nil {
			return nil, RedisParseDSNFailed.Wrap(err)
		}
		if n.nodeConfig.MaxRetries != 0 {
			opt.MaxRetries = n.nodeConfig.MaxRetries
		}

		if n.nodeConfig.TLSInsecure && redisClientUsesTLS(n.nodeConfig.URI, opt) {
			if opt.TLSConfig == nil {
				opt.TLSConfig = &tls.Config{}
			}
			opt.TLSConfig.InsecureSkipVerify = true
		}

		client := redis.NewClient(opt)
		n.client = client
		return n.client, nil
	}

	return n.Shareable.Init(nil, n.nodeConfig.URI, initFunc)
}

func redisClientUsesTLS(uri string, opt *redis.Options) bool {
	if opt != nil && opt.TLSConfig != nil {
		return true
	}
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(uri)), "rediss://")
}

func validateRedisDurationOptions(cfg RedisClientNodeConfiguration) error {
	checks := map[string]string{
		"dialTimeout":     cfg.DialTimeout,
		"readTimeout":     cfg.ReadTimeout,
		"writeTimeout":    cfg.WriteTimeout,
		"minRetryBackoff": cfg.MinRetryBackoff,
		"maxRetryBackoff": cfg.MaxRetryBackoff,
	}
	for name, raw := range checks {
		if raw == "" {
			continue
		}
		if _, err := time.ParseDuration(raw); err != nil {
			return fmt.Errorf("invalid redis %s %q: %w", name, raw, err)
		}
	}
	return nil
}

func applyRedisDurationOption(name string, raw string, set func(time.Duration)) error {
	if raw == "" {
		return nil
	}
	duration, err := time.ParseDuration(raw)
	if err != nil {
		return fmt.Errorf("invalid redis %s %q: %w", name, raw, err)
	}
	set(duration)
	return nil
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
