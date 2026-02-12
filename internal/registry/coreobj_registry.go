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
	"context"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/neohetj/matrix/internal/log"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
)

var (
	// sidFormatRegex enforces the convention of TypeName_Vmajor or TypeName_Vmajor_minor, e.g., EmailContent_V1 or EmailContent_V1_0.
	sidFormatRegex = regexp.MustCompile(`^[A-Z][a-zA-Z0-9]+_V\d+(_\d+)?$`)
)

// DefaultCoreObjRegistry is the default thread-safe implementation of the CoreObjRegistry interface.
type DefaultCoreObjRegistry struct {
	definitions sync.Map
}

// NewCoreObjRegistry creates a new instance of DefaultCoreObjRegistry.
func NewCoreObjRegistry() *DefaultCoreObjRegistry {
	return &DefaultCoreObjRegistry{}
}

// Register adds new object definitions to the registry.
// It also checks if the SID conforms to the recommended format convention.
func (r *DefaultCoreObjRegistry) Register(defs ...types.CoreObjDef) {
	logger := log.GetLogger()
	for _, def := range defs {
		if def == nil {
			continue
		}
		sid := def.SID()
		if !sidFormatRegex.MatchString(sid) {
			// Ignore check for list type
			if !strings.HasPrefix(sid, cnst.LIST_PREFIX) {
				logger.Warnf(context.Background(), "CoreObj SID '%s' does not conform to the recommended format 'TypeName_Vmajor_minor'.", sid)
			}
		}

		// Check for list type naming convention
		obj := def.New()
		if obj != nil {
			t := reflect.TypeOf(obj)
			// Handle pointer to slice
			if t.Kind() == reflect.Pointer {
				t = t.Elem()
			}
			if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
				if !strings.HasPrefix(sid, cnst.LIST_PREFIX) {
					logger.Warnf(context.Background(), "CoreObj SID '%s' is a list type but does not start with '%s'. Recommended format: '[]%s'", sid, cnst.LIST_PREFIX, sid)
				}
			}
		}

		r.definitions.Store(sid, def)
	}
}

// Get retrieves an object definition by its SID.
func (r *DefaultCoreObjRegistry) Get(sid string) (types.CoreObjDef, bool) {
	value, ok := r.definitions.Load(sid)
	if !ok {
		return nil, false
	}
	def, ok := value.(types.CoreObjDef)
	return def, ok
}

// Unregister removes object definitions from the registry.
func (r *DefaultCoreObjRegistry) Unregister(sids ...string) {
	for _, sid := range sids {
		r.definitions.Delete(sid)
	}
}

// GetAll returns all registered object definitions.
func (r *DefaultCoreObjRegistry) GetAll() []types.CoreObjDef {
	var defs []types.CoreObjDef
	r.definitions.Range(func(key, value any) bool {
		if def, ok := value.(types.CoreObjDef); ok {
			defs = append(defs, def)
		}
		return true
	})
	return defs
}
