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

// ChannelManager defines the interface for managing channels in a pipeline.
// It allows external components to register and retrieve channels for communication.
type ChannelManager interface {
	// Register registers a channel with the given pipeline ID and channel name.
	Register(pipelineID, channelName string, ch chan RuleMsg)
	// Unregister removes a channel registration.
	Unregister(pipelineID, channelName string)
	// Get retrieves a registered channel.
	Get(pipelineID, channelName string) (chan RuleMsg, error)
}
