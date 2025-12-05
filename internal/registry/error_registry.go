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

package registry

import (
	"fmt"
	"sync"

	"github.com/NeohetJ/Matrix/pkg/types"
)

// DefaultFaultRegistry is the default thread-safe implementation of the FaultRegistry interface.
type DefaultFaultRegistry struct {
	errors sync.Map
}

// NewFaultRegistry creates a new instance of DefaultFaultRegistry.
func NewFaultRegistry() *DefaultFaultRegistry {
	return &DefaultFaultRegistry{}
}

// Register adds new error definitions to the registry.
// It panics if an error with the same code is already registered.
func (r *DefaultFaultRegistry) Register(errs ...*types.Fault) {
	for _, err := range errs {
		if err != nil {
			if _, loaded := r.errors.LoadOrStore(err.Code, err); loaded {
				panic(fmt.Sprintf("error code %s is already registered", err.Code))
			}
		}
	}
}

// Get retrieves an error definition by its code.
func (r *DefaultFaultRegistry) Get(code int32) (*types.Fault, bool) {
	value, ok := r.errors.Load(code)
	if !ok {
		return nil, false
	}
	err, ok := value.(*types.Fault)
	return err, ok
}
