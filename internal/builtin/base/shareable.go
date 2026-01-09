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

// Package base provides foundational utilities for component development.
package base

import (
	"sync"

	"github.com/neohetj/matrix/pkg/asset"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
)

var (
	NodePoolNil   = &types.Fault{Code: cnst.CodeNodePoolNil, Message: "shared node pool is nil"}
	ClientNotInit = &types.Fault{Code: cnst.CodeClientNotInit, Message: "client not initialized"}
)

// Shareable is a generic helper struct for creating nodes that manage shareable resources.
// It encapsulates the logic for either creating a standalone resource instance or retrieving
// a shared one from a NodePool.
type Shareable[T any] struct {
	// Resource is the Asset reference to the shareable resource.
	Resource asset.Asset[T]

	// InitFunc is the function that creates the actual resource instance as a fallback.
	InitFunc func() (T, error)

	// once ensures lazy initialization happens only once.
	once sync.Once
	// instance holds the resolved resource instance.
	instance T
	// err holds any error that occurred during initialization.
	err error

	// nodePool is the pool to retrieve shared resources from.
	nodePool types.NodePool
}

// Init initializes the Shareable helper.
// It sets up the Asset and NodePool. The actual resolution happens lazily on the first Get() call.
//
// nodePool: The node pool provided by the runtime.
// resourceURI: The URI for the resource (e.g., "ref://my_db", "dsn://mysql/...").
// initFunc: Fallback function to create a new instance if Asset resolution fails or is not applicable.
func (s *Shareable[T]) Init(pool types.NodePool, resourceURI string, initFunc func() (T, error)) error {
	s.nodePool = pool
	s.Resource = asset.Asset[T]{URI: resourceURI}
	s.InitFunc = initFunc
	return nil
}

// Get retrieves the resource instance.
// It lazily resolves the Asset on the first call and caches the result.
func (s *Shareable[T]) Get() (T, error) {
	s.once.Do(func() {
		// 1. Try to resolve via Asset (e.g. ref://)
		if s.nodePool != nil {
			ctx := asset.NewAssetContext(asset.WithNodePool(s.nodePool))
			val, err := s.Resource.Resolve(ctx)
			if err == nil {
				s.instance = val
				return
			}
			// If error is "node pool not found" or "invalid uri scheme", we might want to fallback.
			// But if it's "ref not found", maybe we should fail?
			// Current logic: simple fallback if Resolve fails.
		}

		// 2. Fallback to InitFunc
		if s.InitFunc != nil {
			s.instance, s.err = s.InitFunc()
		} else {
			s.err = ClientNotInit
		}
	})

	return s.instance, s.err
}

// GetInstance provides a generic way to get the instance, satisfying the types.SharedNode interface.
func (s *Shareable[T]) GetInstance() (any, error) {
	return s.Get()
}

// Errors returns the list of possible faults that this node can produce.
func (s *Shareable[T]) Errors() []*types.Fault {
	return []*types.Fault{NodePoolNil, ClientNotInit}
}

// // zeroValue returns the zero value for a generic type T.
// func zeroValue[T any]() T {
// 	var zero T
// 	return zero
// }
