/*
 * Copyright 2025 The Matrix Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package external

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"gitlab.com/neohet/matrix/pkg/components/base"
	"gitlab.com/neohet/matrix/pkg/registry"
	"gitlab.com/neohet/matrix/pkg/types"
	"gitlab.com/neohet/matrix/pkg/utils"
)

const (
	RedisClientNodeType = "external/redisClient"
)

// redisClientNodePrototype is the shared prototype instance for registration.
var redisClientNodePrototype = &RedisClientNode{
	BaseNode: *types.NewBaseNode(RedisClientNodeType, types.NodeDefinition{
		Name:        "Redis Client",
		Description: "Provides a shared Redis client (go-redis).",
		Dimension:   "External",
		Tags:        []string{"external", "redis", "cache"},
		Version:     "1.0.0",
	}),
}

func init() {
	registry.Default.NodeManager.Register(redisClientNodePrototype)
}

// RedisClientNodeConfiguration holds the configuration for the RedisClientNode.
type RedisClientNodeConfiguration struct {
	DSN      string `json:"dsn"`
	PoolSize int    `json:"poolSize"`
}

// RedisClientNode is a component that provides a shared Redis client.
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
		BaseNode: n.BaseNode, // Reference the shared BaseNode template
	}
}

// Init initializes the node.
func (n *RedisClientNode) Init(cfg types.Config) error {
	if err := utils.Decode(cfg, &n.nodeConfig); err != nil {
		return fmt.Errorf("failed to decode redis client node config: %w", err)
	}

	initFunc := func() (*redis.Client, error) {
		if n.client != nil {
			return n.client, nil
		}

		opt, err := redis.ParseURL(n.nodeConfig.DSN)
		if err != nil {
			return nil, fmt.Errorf("failed to parse redis DSN: %w", err)
		}
		if n.nodeConfig.PoolSize > 0 {
			opt.PoolSize = n.nodeConfig.PoolSize
		}
		opt.PoolTimeout = 5 * time.Second

		client := redis.NewClient(opt)
		if err := client.Ping(context.Background()).Err(); err != nil {
			client.Close()
			return nil, fmt.Errorf("failed to connect to redis: %w", err)
		}
		n.client = client
		return n.client, nil
	}

	// The nodePool is injected by the runtime. For now, we pass nil.
	return n.Shareable.Init(nil, n.nodeConfig.DSN, initFunc)
}

// OnMsg is a no-op for this resource node.
func (n *RedisClientNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {
	// No-op
}

// Destroy closes the Redis client connection.
func (n *RedisClientNode) Destroy() {
	n.closeOnce.Do(func() {
		if n.client != nil {
			n.client.Close()
		}
	})
}
