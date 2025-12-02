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

package redis

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gitlab.com/neohet/matrix/pkg/registry"
	"gitlab.com/neohet/matrix/pkg/types"
)

// GetRedisConnection retrieves a Redis connection, either from a shared node pool
// or by creating a new temporary connection.
func GetRedisConnection(nodePool types.NodePool, redisDsn string) (*redis.Client, bool, error) {
	if nodePool == nil {
		nodePool = registry.Default.SharedNodePool
	}

	if strings.HasPrefix(redisDsn, "ref://") {
		nodeId := strings.TrimPrefix(redisDsn, "ref://")
		sharedCtx, ok := nodePool.Get(nodeId)
		if !ok {
			return nil, false, types.ErrNodeNotFound.Wrap(fmt.Errorf("redis client node not found by ref: %s", redisDsn))
		}

		node := sharedCtx.GetNode()
		redisClient, ok := node.(types.SharedNode)
		if !ok {
			return nil, false, types.ErrInternal.Wrap(fmt.Errorf("node %s is not a SharedNode", redisDsn))
		}

		instance, err := redisClient.GetInstance()
		if err != nil {
			return nil, false, types.ErrInternal.Wrap(fmt.Errorf("failed to get redis instance from %s: %w", redisDsn, err))
		}

		client, ok := instance.(*redis.Client)
		if !ok {
			return nil, false, types.ErrInternal.Wrap(fmt.Errorf("shared instance from %s is not a *redis.Client", redisDsn))
		}
		return client, false, nil
	} else {
		opt, err := redis.ParseURL(redisDsn)
		if err != nil {
			return nil, false, types.ErrInternal.Wrap(fmt.Errorf("failed to parse redis DSN: %w", err))
		}
		opt.PoolTimeout = 5 * time.Second
		client := redis.NewClient(opt)
		if err := client.Ping(context.Background()).Err(); err != nil {
			client.Close()
			return nil, false, types.ErrInternal.Wrap(fmt.Errorf("failed to connect to redis: %w", err))
		}
		return client, true, nil
	}
}
