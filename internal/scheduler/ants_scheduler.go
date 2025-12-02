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

package scheduler

import "github.com/panjf2000/ants/v2"

// AntsScheduler is an implementation of the Scheduler interface using the ants goroutine pool.
type AntsScheduler struct {
	pool *ants.Pool
}

// NewAntsScheduler creates a new scheduler with a specified pool size.
func NewAntsScheduler(size int) (*AntsScheduler, error) {
	pool, err := ants.NewPool(size)
	if err != nil {
		return nil, err
	}
	return &AntsScheduler{pool: pool}, nil
}

// Submit submits a task to the ants pool for execution.
func (s *AntsScheduler) Submit(task func()) error {
	return s.pool.Submit(task)
}

// Stop gracefully releases the ants pool.
func (s *AntsScheduler) Stop() {
	s.pool.Release()
}
