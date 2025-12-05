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

package contract

import (
	"encoding/json"
	"time"

	"github.com/NeohetJ/Matrix/pkg/cnst"
	"github.com/NeohetJ/Matrix/pkg/types"
	"github.com/google/uuid"
)

// DefaultRuleMsg is the default implementation of the RuleMsg interface.
type DefaultRuleMsg struct {
	id         string
	ts         int64
	msgType    string
	dataFormat cnst.MFormat
	data       types.Data
	dataT      types.DataT
	metadata   types.Metadata
}

// NewDefaultRuleMsg creates a new message instance.
func NewDefaultRuleMsg(msgType, data string, metadata types.Metadata, dataT types.DataT) *DefaultRuleMsg {
	if metadata == nil {
		metadata = make(types.Metadata)
	}
	return &DefaultRuleMsg{
		id:         uuid.NewString(),
		ts:         time.Now().UnixMilli(),
		msgType:    msgType,
		data:       types.Data(data),
		dataT:      dataT,
		metadata:   metadata,
		dataFormat: cnst.TEXT,
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

func (m *DefaultRuleMsg) DataFormat() cnst.MFormat {
	return m.dataFormat
}

func (m *DefaultRuleMsg) Data() types.Data {
	return m.data
}

func (m *DefaultRuleMsg) DataT() types.DataT {
	return m.dataT
}

func (m *DefaultRuleMsg) Metadata() types.Metadata {
	return m.metadata
}

func (m *DefaultRuleMsg) SetData(data string, format cnst.MFormat) {
	m.data = types.Data(data)
	m.WithDataFormat(format)
}

func (m *DefaultRuleMsg) SetMetadata(metadata types.Metadata) {
	m.metadata = metadata
}

func (m *DefaultRuleMsg) WithDataFormat(format cnst.MFormat) types.RuleMsg {
	if format.IsValid() {
		m.dataFormat = format
	} else {
		m.dataFormat = cnst.UNKNOWN
	}
	return m
}

func (m *DefaultRuleMsg) Copy() types.RuleMsg {
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
func (m *DefaultRuleMsg) DeepCopy() (types.RuleMsg, error) {
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
		Id         string         `json:"id"`
		Ts         int64          `json:"ts"`
		MsgType    string         `json:"msgType"`
		DataFormat cnst.MFormat   `json:"dataFormat"`
		Data       types.Data     `json:"data"`
		DataT      types.DataT    `json:"dataT"`
		Metadata   types.Metadata `json:"metadata"`
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
