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
	"sync"

	"gitlab.com/neohet/matrix/pkg/utils"
)

// DefaultDataT is the default thread-safe implementation of the DataT interface.
type DefaultDataT struct {
	mu   sync.RWMutex
	data map[string]CoreObj
}

// NewDataT creates a new instance of DefaultDataT.
func NewDataT() *DefaultDataT {
	return &DefaultDataT{
		data: make(map[string]CoreObj),
	}
}

// Get retrieves a business object by its unique object ID (objId).
func (d *DefaultDataT) Get(objId string) (CoreObj, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	item, ok := d.data[objId]
	return item, ok
}

// Set adds or updates a business object in the container using its objId as the key.
func (d *DefaultDataT) Set(objId string, value CoreObj) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.data[objId] = value
}

// GetAll returns a map of all business objects.
func (d *DefaultDataT) GetAll() map[string]CoreObj {
	d.mu.RLock()
	defer d.mu.RUnlock()

	copiedData := make(map[string]CoreObj, len(d.data))
	for k, v := range d.data {
		copiedData[k] = v
	}
	return copiedData
}

// NewItem creates a new CoreObj instance based on a registered definition (SID),
// assigns it the given object ID (objId), and adds it to the container.
func (d *DefaultDataT) NewItem(sid, objId string) (CoreObj, error) {
	if DefaultRegistry == nil {
		panic("DefaultRegistry is not initialized. The registry package must be imported.")
	}
	registry := DefaultRegistry.GetCoreObjRegistry()

	def, ok := registry.Get(sid)
	if !ok {
		return nil, fmt.Errorf("CoreObj definition with sid='%s' not found in registry", sid)
	}

	newItem := newCoreObj(objId, def)
	d.Set(objId, newItem)
	return newItem, nil
}

// Copy returns a deep copy of the container.
func (d *DefaultDataT) Copy() DataT {
	d.mu.RLock()
	defer d.mu.RUnlock()

	newData := &DefaultDataT{
		data: make(map[string]CoreObj, len(d.data)),
	}
	for k, v := range d.data {
		// TODO: Implement a proper deep copy for values if they are pointers or complex types.
		newData.data[k] = v
	}
	return newData
}

// DeepCopy performs a full deep copy of the DataT container and its contents.
// It iterates through all CoreObj items and calls their DeepCopy method.
func (d *DefaultDataT) DeepCopy() (DataT, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	newData := &DefaultDataT{
		data: make(map[string]CoreObj, len(d.data)),
	}

	for k, v := range d.data {
		copiedObj, err := v.DeepCopy()
		if err != nil {
			return nil, fmt.Errorf("failed to deep copy item '%s': %w", k, err)
		}
		newData.data[k] = copiedObj
	}

	return newData, nil
}

// MarshalJSON implements the json.Marshaler interface for DefaultDataT.
// It ensures that the map of CoreObjs is correctly serialized.
func (d *DefaultDataT) MarshalJSON() ([]byte, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return json.Marshal(d.data)
}

// GetByParam retrieves a business object by its logical parameter name.
func (d *DefaultDataT) GetByParam(ctx NodeCtx, pname string) (CoreObj, error) {
	inputs, ok := GetInputs(ctx)
	if !ok {
		return nil, fmt.Errorf("inputs not found in node configuration for node %s", ctx.NodeID())
	}
	objId, err := ResolveParamKey(pname, inputs)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve param '%s' for node %s: %w", pname, ctx.NodeID(), err)
	}
	if obj, ok := d.Get(objId); ok {
		return obj, nil
	} else {
		return nil, fmt.Errorf("pname %s, obj %s not found in dataT for node %s", pname, objId, ctx.NodeID())
	}
}

// NewItemByParam creates a new business object by its logical parameter name.
func (d *DefaultDataT) NewItemByParam(ctx NodeCtx, pname string) (CoreObj, error) {
	outputs, ok := GetOutputs(ctx)
	if !ok {
		return nil, fmt.Errorf("outputs not found in node configuration for node %s", ctx.NodeID())
	}
	config, ok := outputs[pname]
	if !ok {
		return nil, fmt.Errorf("parameter '%s' not found in outputs configuration for node %s", pname, ctx.NodeID())
	}
	if config.ObjId == "" {
		return nil, fmt.Errorf("objId is empty for parameter '%s' in node %s", pname, ctx.NodeID())
	}
	if config.DefineSID == "" {
		return nil, fmt.Errorf("defineSid is empty for parameter '%s' in node %s", pname, ctx.NodeID())
	}

	return d.NewItem(config.DefineSID, config.ObjId)
}

// IOConfig defines the structure for a single input/output item in a node's configuration.
type IOConfig struct {
	ObjId     string `json:"objId"`
	DefineSID string `json:"defineSid"`
}

// ResolveParamKey finds the corresponding objId for a given parameter name (pname)
// from a map of IOConfigs.
func ResolveParamKey(pname string, ioConfigs map[string]IOConfig) (string, error) {
	if ioConfigs == nil {
		return "", fmt.Errorf("IO configuration map is nil")
	}
	config, ok := ioConfigs[pname]
	if !ok {
		return "", fmt.Errorf("parameter '%s' not found in IO configuration", pname)
	}
	if config.ObjId == "" {
		return "", fmt.Errorf("objId is empty for parameter '%s'", pname)
	}
	return config.ObjId, nil
}

// getIOConfigMap extracts and decodes an IO map (inputs or outputs) from the node context.
func getIOConfigMap(ctx NodeCtx, mapKey string) (map[string]IOConfig, bool) {
	nodeDef := ctx.SelfDef()
	if nodeDef == nil {
		return nil, false
	}

	var ioMapRaw map[string]any
	var ok bool

	switch mapKey {
	case "inputs":
		ioMapRaw, ok = nodeDef.Inputs, true
	case "outputs":
		ioMapRaw, ok = nodeDef.Outputs, true
	}

	if !ok || ioMapRaw == nil {
		return nil, false
	}

	// Use the utils.Decode function to convert map[string]any to the target map.
	var ioConfigs map[string]IOConfig
	if err := utils.Decode(ioMapRaw, &ioConfigs); err != nil {
		if logger := ctx.Logger(); logger != nil {
			logger.Warnf(ctx.GetContext(), "failed to decode '%s' map: %v", mapKey, err)
		}
		return nil, false
	}

	return ioConfigs, true
}

// GetInputs extracts the inputs configuration from a node's context.
func GetInputs(ctx NodeCtx) (map[string]IOConfig, bool) {
	return getIOConfigMap(ctx, "inputs")
}

// GetOutputs extracts the outputs configuration from a node's context.
func GetOutputs(ctx NodeCtx) (map[string]IOConfig, bool) {
	return getIOConfigMap(ctx, "outputs")
}
