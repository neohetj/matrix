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
	"fmt"
)

// defaultCoreObj is the default implementation of the CoreObj interface.
type defaultCoreObj struct {
	key  string
	def  *CoreObjDef
	body any
}

// NewCoreObj creates a new instance of the default CoreObj implementation.
// This function is not exported, as CoreObjs should be created via DataT.NewItem().
func newCoreObj(key string, def *CoreObjDef) CoreObj {
	return &defaultCoreObj{
		key:  key,
		def:  def,
		body: def.New(),
	}
}

func (d *defaultCoreObj) Key() string {
	return d.key
}

func (d *defaultCoreObj) Definition() *CoreObjDef {
	return d.def
}

func (d *defaultCoreObj) Body() any {
	return d.body
}

func (d *defaultCoreObj) SetBody(body any) error {
	// A real implementation might perform type checking here.
	// For now, we trust the caller.
	if body == nil {
		return fmt.Errorf("body cannot be nil")
	}
	d.body = body
	return nil
}

// DeepCopy creates a deep copy of the CoreObj instance.
// It handles basic types by direct assignment and complex types by JSON marshaling/unmarshaling.
func (d *defaultCoreObj) DeepCopy() (CoreObj, error) {
	if d.body == nil {
		return &defaultCoreObj{
			key:  d.key,
			def:  d.def,
			body: nil,
		}, nil
	}

	// For basic types, direct assignment is sufficient for a deep copy.
	// This avoids unnecessary JSON marshaling/unmarshaling errors for primitive types.
	switch d.body.(type) {
	case string, bool,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64,
		json.Number: // json.Number is also considered a basic type for direct copy
		return &defaultCoreObj{
			key:  d.key,
			def:  d.def,
			body: d.body,
		}, nil
	}

	// For complex types (structs, maps, slices), use JSON marshaling/unmarshaling for deep copy.
	bodyBytes, err := json.Marshal(d.body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal core object body for deep copy: %w", err)
	}

	// Create a new instance of the target type.
	newBody := d.def.New()

	// Unmarshal the JSON bytes into the new instance.
	if err := json.Unmarshal(bodyBytes, newBody); err != nil {
		return nil, fmt.Errorf("failed to unmarshal core object body for deep copy: %w", err)
	}

	// Create the new CoreObj with the deep-copied body.
	newObj := &defaultCoreObj{
		key:  d.key,
		def:  d.def, // Definition is shared, it's immutable.
		body: newBody,
	}
	return newObj, nil
}

// MarshalJSON implements the json.Marshaler interface for defaultCoreObj.
// It ensures that the object's body is correctly serialized, not the object's internal fields.
func (d *defaultCoreObj) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.body)
}
