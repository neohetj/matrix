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

// Package scheduler defines the interface for task scheduling and concurrency management.
// Author: Neohet
package types

// Scheduler is the interface for a task scheduler.
// It is responsible for managing a pool of goroutines to execute tasks asynchronously.
type Scheduler interface {
	// Submit submits a task to the scheduler for execution.
	// It returns an error if the scheduler is closed or the task cannot be accepted.
	Submit(task func()) error

	// Stop gracefully shuts down the scheduler, waiting for all active tasks to complete.
	Stop()
}
