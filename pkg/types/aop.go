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

package types

// Aspect is an interface for components that can intercept the execution of a node.
// Aspects are designed to be stateless and can be executed in parallel.
// They are suitable for implementing cross-cutting concerns like logging, metrics,
// security checks, or request/response manipulation.
type Aspect interface {
	// Before is executed before a node's OnMsg method is called.
	// It can modify the message before the node processes it.
	// If it returns an error, the node's OnMsg will be skipped, and the error
	// will be passed to the After method.
	Before(ctx NodeCtx, msg RuleMsg) (RuleMsg, error)

	// After is executed after a node's OnMsg method completes.
	// It receives the original message and any error that occurred during
	// the node's execution (or from the Before method).
	After(ctx NodeCtx, msg RuleMsg, err error)
}

// CallbackFunc is an interface for components that process aggregated results
// from a rule chain execution. Callbacks are stateful and are executed serially.
// They are ideal for scenarios that require a complete picture of the execution,
// such as generating a final report, snapshotting the entire run, or
// broadcasting detailed execution logs.
type CallbackFunc interface {
	// OnNodeCompleted is called serially every time a node finishes its execution.
	OnNodeCompleted(ctx NodeCtx, msg RuleMsg, err error)

	// OnChainCompleted is called once at the very end of the entire rule chain execution.
	// It receives the final message and error of the chain.
	OnChainCompleted(msg RuleMsg, err error)
}
