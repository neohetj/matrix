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

// Package types defines the core data structures and interfaces for the Matrix rule engine.
package types

import "context"

// Config is a type for component configurations, represented as a map with string keys and any values.
type Config map[string]any

// Metadata is a type for message metadata, represented as a map with string keys and string values.
type Metadata map[string]string

// DataType defines the type of data, used for validation and conversion.
type DataType string

// DataFormat defines the format of the Data field in a RuleMsg.
type DataFormat string

// Copy creates a new copy of the metadata.
func (m Metadata) Copy() Metadata {
	newMd := make(Metadata, len(m))
	for k, v := range m {
		newMd[k] = v
	}
	return newMd
}

// DataT is the interface for a container of structured business objects.
// It holds multiple, typed objects, accessible by a unique key.
type DataT interface {
	// Get retrieves a business object by its unique object ID (objId).
	Get(objId string) (CoreObj, bool)
	// Set adds or updates a business object in the container using its objId as the key.
	Set(objId string, value CoreObj)
	// NewItem creates a new CoreObj instance based on a registered definition (SID),
	// assigns it the given object ID (objId), and adds it to the container.
	NewItem(sid, objId string) (CoreObj, error)
	// GetAll returns a map of all business objects.
	GetAll() map[string]CoreObj
	// Copy returns a shallow copy of the container.
	Copy() DataT
	// DeepCopy returns a deep copy of the container.
	DeepCopy() (DataT, error)

	// GetByParam retrieves a business object by its logical parameter name,
	// by resolving the name to an objId using the node's context.
	// It returns (nil, nil) if the object is not found but no error occurred.
	GetByParam(ctx NodeCtx, pname string) (CoreObj, error)

	// NewItemByParam creates a new business object by its logical parameter name,
	// by resolving the name to an objId and SID using the node's context.
	NewItemByParam(ctx NodeCtx, pname string) (CoreObj, error)
}

// RuleMsg is the interface for a message in the rule engine.
// It carries both raw, serialized data and structured, typed business objects.
type RuleMsg interface {
	// ID returns the message ID.
	ID() string
	// Ts returns the message timestamp.
	Ts() int64
	// Type returns the message type.
	Type() string
	// DataFormat returns the format of the Data field, e.g., "JSON", "TEXT".
	DataFormat() DataFormat
	// WithDataFormat sets the format of the Data field and returns the message for chaining.
	WithDataFormat(dataFormat DataFormat) RuleMsg
	// Data returns the raw message payload, typically a JSON string.
	// This is used for communication with external systems.
	Data() string
	// DataT returns the container for structured business objects.
	// This is used for processing within the rule chain.
	DataT() DataT
	// Metadata returns the message metadata.
	Metadata() Metadata
	// SetData sets the raw message payload.
	SetData(data string)
	// SetMetadata sets the message metadata.
	SetMetadata(metadata Metadata)
	// Copy returns a copy of the message.
	Copy() RuleMsg
	// DeepCopy returns a deep copy of the message.
	DeepCopy() (RuleMsg, error)
}

// Logger is the interface for a logger.
type Logger interface {
	// Printf is used for compatibility with the standard library logger.
	Printf(ctx context.Context, format string, v ...any)
	// Debugf logs a message at debug level.
	Debugf(ctx context.Context, format string, v ...any)
	// Infof logs a message at info level.
	Infof(ctx context.Context, format string, v ...any)
	// Warnf logs a message at warning level.
	Warnf(ctx context.Context, format string, v ...any)
	// Errorf logs a message at error level.
	Errorf(ctx context.Context, format string, v ...any)
	// With returns a new logger instance with the given key-value pairs.
	With(fields ...any) Logger
}
