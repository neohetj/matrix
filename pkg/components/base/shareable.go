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
	"errors"
	"strings"
	"sync"

	"gitlab.com/neohet/matrix/pkg/types"
)

const (
	// RefPrefix is the prefix for referencing a shared node instance from the node pool.
	RefPrefix = "ref://"
)

var (
	ErrNodePoolNil   = errors.New("node pool is nil")
	ErrClientNotInit = errors.New("client not initialized")
)

// Shareable is a generic helper struct for creating nodes that manage shareable resources.
// It encapsulates the logic for either creating a standalone resource instance or retrieving
// a shared one from a NodePool.
type Shareable[T any] struct {
	// Locker protects the initialization of the resource.
	Locker sync.Mutex
	// InitFunc is the function that creates the actual resource instance.
	InitFunc func() (T, error)
	// instance holds the created resource instance.
	instance any
	// nodePool is the pool to retrieve shared resources from.
	nodePool types.NodePool
	// instanceId is the ID of the resource in the node pool.
	instanceId string
	// isFromPool indicates whether the resource is retrieved from the pool.
	isFromPool bool
}

// Init initializes the Shareable helper.
// It determines whether to use a shared resource from the pool or to prepare for creating a standalone one.
//
// nodePool: The node pool provided by the runtime.
// resourcePath: The configuration path for the resource, e.g., a DSN or a reference like "ref://my_db".
// initFunc: The function to call to create a new instance if not using the pool.
func (s *Shareable[T]) Init(pool types.NodePool, resourcePath string, initFunc func() (T, error)) error {
	s.nodePool = pool
	s.InitFunc = initFunc

	if strings.HasPrefix(resourcePath, RefPrefix) {
		// This is a reference to a shared resource.
		s.isFromPool = true
		s.instanceId = strings.TrimPrefix(resourcePath, RefPrefix)
	}
	// For non-ref paths, we do nothing here. The instance will be created on the first Get() call.
	return nil
}

// Get retrieves the resource instance.
// If the resource is from the pool, it's fetched from there.
// If it's a standalone resource, it's created on the first call (lazy initialization).
func (s *Shareable[T]) Get() (T, error) {
	if s.isFromPool {
		// Retrieve from the node pool.
		if s.nodePool == nil {
			return zeroValue[T](), ErrNodePoolNil
		}
		instance, err := s.nodePool.GetInstance(s.instanceId)
		if err != nil {
			return zeroValue[T](), err
		}
		return instance.(T), nil
	}

	// Standalone instance: check if it's already initialized.
	s.Locker.Lock()
	defer s.Locker.Unlock()

	// Double-check locking pattern.
	if s.instance != nil {
		return s.instance.(T), nil
	}

	if s.InitFunc == nil {
		return zeroValue[T](), ErrClientNotInit
	}

	// Initialize the instance.
	newInstance, err := s.InitFunc()
	if err != nil {
		return zeroValue[T](), err
	}
	s.instance = newInstance
	return newInstance, nil
}

// GetInstance provides a generic way to get the instance, satisfying the types.SharedNode interface.
func (s *Shareable[T]) GetInstance() (interface{}, error) {
	return s.Get()
}

// IsFromPool returns true if the resource is configured to be retrieved from a pool.
func (s *Shareable[T]) IsFromPool() bool {
	return s.isFromPool
}

// zeroValue returns the zero value for a generic type T.
func zeroValue[T any]() T {
	var zero T
	return zero
}
