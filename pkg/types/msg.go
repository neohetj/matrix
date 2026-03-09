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

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/neohetj/matrix/pkg/cnst"
)

// CoreObjDef holds the definition of a business object, including its OpenAPI schema.
type CoreObjDef interface {
	json.Marshaler

	// SID returns the unique type identifier for the object definition.
	SID() string

	// New returns a new instance of the business object.
	New() any

	// Description returns a human-readable description of the object.
	Description() string

	// Schema returns the JSON schema of the object's fields in OpenAPI format.
	Schema() string

	// OpenAPISchema returns the raw openapi3.Schema object.
	OpenAPISchema() *openapi3.Schema
}

// CoreObj is the interface for a concrete business object instance.
type CoreObj interface {
	Key() string
	Definition() CoreObjDef
	Body() any
	SetBody(body any) error
	DeepCopy() (CoreObj, error)
}

// CoreObjRegistry is the interface for managing a collection of CoreObjDef.
type CoreObjRegistry interface {
	Register(defs ...CoreObjDef)
	Get(sid string) (CoreObjDef, bool)
	GetAll() []CoreObjDef
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
	// Project returns a new container that only retains the specified object IDs.
	// The original container is left untouched.
	Project(keepObjIDs []string) (DataT, error)

	// GetByParam retrieves a business object by its logical parameter name,
	// by resolving the name to an objId using the node's context.
	// It returns (nil, nil) if the object is not found but no error occurred.
	GetByParam(ctx NodeCtx, pname string) (CoreObj, error)

	// NewItemByParam creates a new business object by its logical parameter name,
	// by resolving the name to an objId and SID using the node's context.
	NewItemByParam(ctx NodeCtx, pname string) (CoreObj, error)
}

// Data is a type for message data, represented as a string.
type Data string

// Metadata is a type for message metadata, represented as a map with string keys and string values.
type Metadata map[string]string

// Copy creates a new copy of the metadata.
func (m Metadata) Copy() Metadata {
	newMd := make(Metadata, len(m))
	for k, v := range m {
		newMd[k] = v
	}
	return newMd
}

// Config is a type for component configurations, represented as a map with string keys and any values.
type ConfigMap map[string]any

// Get retrieves a value from the ConfigMap by key and casts it to type T.
// It returns the zero value of T and false if the key is missing or the type assertion fails.
func GetConfigMap[T any](c ConfigMap, key string) (T, bool) {
	val, ok := c[key]
	if !ok {
		var zero T
		return zero, false
	}
	tVal, ok := val.(T)
	if ok {
		return tVal, true
	}

	// Try ConfigMap conversion if T is ConfigMap and val is map[string]any
	if _, isConfigMap := any(tVal).(ConfigMap); isConfigMap {
		if mVal, isMap := val.(map[string]any); isMap {
			return any(ConfigMap(mVal)).(T), true
		}
	}

	return tVal, false
}

// Merge merges another ConfigMap into this one.
// Keys in the other ConfigMap will overwrite keys in this ConfigMap.
func (c ConfigMap) Merge(other ConfigMap) {
	for k, v := range other {
		c[k] = v
	}
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
	DataFormat() cnst.MFormat
	// Data returns the raw message payload, typically a JSON string.
	// This is used for communication with external systems.
	Data() Data
	// DataT returns the container for structured business objects.
	// This is used for processing within the rule chain.
	DataT() DataT
	// Metadata returns the message metadata.
	Metadata() Metadata

	// SetData sets the raw message payload and its format.
	SetData(data string, format cnst.MFormat)
	// SetMetadata sets the message metadata.
	SetMetadata(metadata Metadata)

	// Copy returns a copy of the message.
	Copy() RuleMsg
	// DeepCopy returns a deep copy of the message.
	DeepCopy() (RuleMsg, error)

	// WithDataFormat sets the raw message payload format and returns the message.
	WithDataFormat(format cnst.MFormat) RuleMsg
}

// RuleMsgDataTCloner is implemented by RuleMsg types that can preserve message identity
// while replacing the structured DataT payload.
type RuleMsgDataTCloner interface {
	CloneWithDataT(dataT DataT) RuleMsg
}
