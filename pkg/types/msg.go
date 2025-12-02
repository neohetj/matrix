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

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// DefaultRuleMsg is the default implementation of the RuleMsg interface.
type DefaultRuleMsg struct {
	id         string
	ts         int64
	msgType    string
	dataFormat DataFormat
	data       string
	dataT      DataT
	metadata   Metadata
}

// NewMsg creates a new message with the given type, data, and metadata.
// It automatically generates a new UUID and sets the timestamp.
// The dataFormat is initially empty and should be set explicitly via WithDataFormat.
func NewMsg(msgType, data string, metadata Metadata, dataT DataT) RuleMsg {
	if metadata == nil {
		metadata = make(Metadata)
	}
	if dataT == nil {
		dataT = NewDataT()
	}
	return &DefaultRuleMsg{
		id:         uuid.NewString(),
		ts:         time.Now().UnixMilli(),
		msgType:    msgType,
		data:       data,
		dataT:      dataT,
		metadata:   metadata,
		dataFormat: "", // Default to empty
	}
}

func (m *DefaultRuleMsg) ID() string {
	return m.id
}

func (m *DefaultRuleMsg) Ts() int64 {
	return m.ts
}

func (m *DefaultRuleMsg) Type() string {
	return m.msgType
}

func (m *DefaultRuleMsg) DataFormat() DataFormat {
	return m.dataFormat
}

func (m *DefaultRuleMsg) WithDataFormat(dataFormat DataFormat) RuleMsg {
	m.dataFormat = dataFormat
	return m
}

func (m *DefaultRuleMsg) Data() string {
	return m.data
}

func (m *DefaultRuleMsg) DataT() DataT {
	return m.dataT
}

func (m *DefaultRuleMsg) Metadata() Metadata {
	return m.metadata
}

func (m *DefaultRuleMsg) SetData(data string) {
	m.data = data
}

func (m *DefaultRuleMsg) SetMetadata(metadata Metadata) {
	m.metadata = metadata
}

func (m *DefaultRuleMsg) Copy() RuleMsg {
	// Create a new message instance.
	newMsg := &DefaultRuleMsg{
		// Keep the original ID and timestamp to trace the message origin.
		id:         m.id,
		ts:         m.ts,
		msgType:    m.msgType,
		dataFormat: m.dataFormat,
		data:       m.data,
		// Shallow copy of DataT (share the reference) as per design.
		dataT: m.dataT,
		// Deep copy of Metadata (each branch gets its own metadata) to prevent race conditions.
		metadata: m.metadata.Copy(),
	}
	return newMsg
}

// DeepCopy creates a full, deep copy of the RuleMsg.
func (m *DefaultRuleMsg) DeepCopy() (RuleMsg, error) {
	// Deep copy the DataT container.
	newDataT, err := m.dataT.DeepCopy()
	if err != nil {
		return nil, err
	}

	// Create a new message instance with the deep-copied DataT.
	newMsg := &DefaultRuleMsg{
		id:         m.id,
		ts:         m.ts,
		msgType:    m.msgType,
		dataFormat: m.dataFormat,
		data:       m.data,
		dataT:      newDataT,
		metadata:   m.metadata.Copy(), // Metadata's Copy is a deep copy.
	}
	return newMsg, nil
}

// MarshalJSON implements the json.Marshaler interface.
// This allows us to control the serialization of the DefaultRuleMsg,
// ensuring private fields are included in the JSON output.
func (m *DefaultRuleMsg) MarshalJSON() ([]byte, error) {
	// Use a temporary struct with public fields for marshaling.
	return json.Marshal(&struct {
		Id         string     `json:"id"`
		Ts         int64      `json:"ts"`
		MsgType    string     `json:"msgType"`
		DataFormat DataFormat `json:"dataFormat"`
		Data       string     `json:"data"`
		DataT      DataT      `json:"dataT"`
		Metadata   Metadata   `json:"metadata"`
	}{
		Id:         m.id,
		Ts:         m.ts,
		MsgType:    m.msgType,
		DataFormat: m.dataFormat,
		Data:       m.data,
		DataT:      m.dataT,
		Metadata:   m.metadata,
	})
}
