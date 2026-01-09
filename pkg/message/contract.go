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

package message

import (
	"github.com/neohetj/matrix/internal/contract"
	"github.com/neohetj/matrix/pkg/types"
)

// NewDataT creates a new instance of the default DataT implementation.
func NewDataT() types.DataT {
	// TODO: 支持使用第三方定义的DataT
	return contract.NewDataT()
}

// NewCoreObj creates a new instance of the default CoreObj implementation.
func NewCoreObj(key string, def types.CoreObjDef) types.CoreObj {
	return contract.NewDefaultCoreObj(key, def)
}

// NewCoreObj creates a new instance of the default CoreObj implementation.
func NewCoreObjDef(prototype any, sid, desc string) types.CoreObjDef {
	return contract.NewDefaultCoreObjDef(prototype, sid, desc)
}
