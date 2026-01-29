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

package action

import (
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/neohetj/matrix/internal/contract"
	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/types"
)

const AggregatorNodeType = "action/aggregator"

func init() {
	registry.Default.NodeManager.Register(aggregatorNodePrototype)
}

var aggregatorNodePrototype = &AggregatorNode{
	BaseNode: *types.NewBaseNode(AggregatorNodeType, types.NodeMetadata{
		Name:        "Aggregator",
		Description: "Waits for messages from all upstream nodes before proceeding. Acts as a parallel join.",
		Dimension:   "Flow Control",
		Tags:        []string{"flow", "aggregator", "join"},
		Version:     "1.0.0",
		NodeReads:   []types.ContractDef{},
		NodeWrites:  []types.ContractDef{},
	}),
}

// AggregatorNode waits for messages from all upstream nodes before proceeding.
type AggregatorNode struct {
	types.BaseNode
	types.Instance
}

// aggState holds the list of received node IDs.
type aggState struct {
	mu        sync.Mutex
	stopChan  chan struct{}
	Received  []string `json:"received"`
	Completed bool     `json:"completed"`
	TimedOut  bool     `json:"timedOut"`
}

// New returns a new instance of the node.
func (n *AggregatorNode) New() types.Node {
	return &AggregatorNode{
		BaseNode: n.BaseNode,
	}
}

// Init initializes the node.
func (n *AggregatorNode) Init(configuration types.ConfigMap) error {
	return nil
}

// Type returns the node type.
func (n *AggregatorNode) Type() types.NodeType {
	return AggregatorNodeType
}

// OnMsg processes the incoming message.
func (n *AggregatorNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {
	senderID := ctx.PreviousNodeID()
	needed := n.getPredecessors(ctx)

	// If no predecessors (start node?), pass through immediately.
	if len(needed) == 0 {
		ctx.TellSuccess(msg)
		return
	}

	stateKey := fmt.Sprintf("agg_state_%s", n.ID())
	var state *aggState
	isNewState := false

	// Retrieve or initialize state from DataT
	stateObj, found := msg.DataT().Get(stateKey)
	if !found {
		state = &aggState{
			Received: []string{},
			stopChan: make(chan struct{}),
		}
		isNewState = true

		def := contract.NewDefaultCoreObjDef(&aggState{}, "aggState", "Aggregator Node State")
		stateObj = contract.NewDefaultCoreObj(stateKey, def)

		if err := stateObj.SetBody(state); err != nil {
			ctx.HandleError(msg, fmt.Errorf("failed to set state body: %w", err))
			return
		}

		msg.DataT().Set(stateKey, stateObj)
	} else {
		var ok bool
		state, ok = stateObj.Body().(*aggState)
		if !ok {
			ctx.HandleError(msg, fmt.Errorf("invalid state type in DataT for key %s", stateKey))
			return
		}
	}

	// State Locking
	state.mu.Lock()
	defer state.mu.Unlock()

	// If already completed or timed out, ignore late messages
	if state.Completed || state.TimedOut {
		// Just acknowledge completion to predecessor without triggering downstream
		ctx.TellNext(msg, "Wait")
		return
	}

	// Start Timeout Timer on first message
	if isNewState {
		timeoutStr, _ := types.GetConfigMap[string](ctx.Config(), "timeout")
		if timeoutStr != "" {
			if duration, err := time.ParseDuration(timeoutStr); err == nil {
				// We must use a copy of the message for the failure callback to avoid race conditions/mutation issues.
				msgCopy := msg.Copy()

				go func() {
					select {
					case <-time.After(duration):
						state.mu.Lock()
						defer state.mu.Unlock()

						if !state.Completed && !state.TimedOut {
							state.TimedOut = true
							ctx.TellFailure(msgCopy, fmt.Errorf("aggregator timed out after %s", timeoutStr))
						}
					case <-ctx.GetContext().Done():
						// Context finished, stop timer
					case <-state.stopChan:
						// Aggregation completed successfully, stop timer
					}
				}()
			} else {
				ctx.Warn(fmt.Sprintf("Invalid timeout duration: %s", timeoutStr))
			}
		}
	}

	// Add sender to received list
	if senderID != "" && !slices.Contains(state.Received, senderID) {
		state.Received = append(state.Received, senderID)
	}

	// Check if all needed messages are received
	if len(state.Received) >= len(needed) {
		state.Completed = true

		// Signal timer to stop
		if state.stopChan != nil {
			close(state.stopChan)
		}

		// Reset received list (optional cleanup)
		state.Received = []string{}

		ctx.TellSuccess(msg)
	} else {
		// Acknowledge receipt to predecessor, but hold flow.
		// "Wait" is a dummy relation. TellNext calls childDone if relation not found.
		ctx.TellNext(msg, "Wait")
	}
}

// getPredecessors finds all nodes that have a connection to the current node.
func (n *AggregatorNode) getPredecessors(ctx types.NodeCtx) []string {
	var predecessors []string
	myID := ctx.NodeID()

	runtime := ctx.GetRuntime()
	if runtime == nil {
		return nil
	}
	chain := runtime.GetChainInstance()
	if chain == nil {
		return nil
	}
	def := chain.Definition()
	if def == nil {
		return nil
	}

	for _, conn := range def.Metadata.Connections {
		if conn.ToID == myID {
			predecessors = append(predecessors, conn.FromID)
		}
	}
	return predecessors
}
