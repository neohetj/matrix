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

package runtime

// import (
// 	"context"
// 	"os"
// 	"testing"

// 	"github.com/stretchr/testify/assert"

// 	"github.com/NeohetJ/Matrix/internal/parser"
// 	"github.com/NeohetJ/Matrix/internal/scheduler"
// 	"github.com/NeohetJ/Matrix/internal/registry"
// 	"github.com/NeohetJ/Matrix/pkg/types"

// 	_ "github.com/NeohetJ/Matrix/pkg/components/action"
// 	_ "github.com/NeohetJ/Matrix/pkg/components/external"
// 	_ "github.com/NeohetJ/Matrix/test/common/nodes"
// )

// func TestRuntime_WithNodePool(t *testing.T) {
// 	// 1. Load shared nodes into the global DefaultNodePool.
// 	// The required node components (DBClientNode, etc.) are self-registered
// 	// via blank imports above. We use the global default NodeManager.
// 	sharedDSL, err := os.ReadFile("../../test/rulechains/shared_pool.json")
// 	assert.NoError(t, err)

// 	_, err = registry.Default.SharedNodePool.Load(sharedDSL, registry.Default.NodeManager)
// 	assert.NoError(t, err)

// 	// Verify that the shared node is in the pool
// 	sharedDB, ok := registry.Default.SharedNodePool.Get("my_db")
// 	assert.True(t, ok)
// 	assert.NotNil(t, sharedDB)

// 	// 2. Load the business rule chain
// 	businessDSL, err := os.ReadFile("../../test/rulechains/business_chain.json")
// 	assert.NoError(t, err)

// 	p := &parser.JsonParser{}
// 	chainDef, err := p.DecodeRuleChain(businessDSL)
// 	assert.NoError(t, err)

// 	// 3. Create a new runtime. It will use the default global NodeManager and NodePool.
// 	s, _ := scheduler.NewAntsScheduler(10)
// 	defer s.Stop()

// 	runtime, err := NewDefaultRuntime(s, chainDef)
// 	assert.NoError(t, err)
// 	defer runtime.Destroy()

// 	// 4. Create a simple message to execute
// 	msg := types.NewMsg("USER_CHECK_REQUEST", `{"user":"test"}`, nil, nil).WithDataFormat(types.JSON)

// 	// 5. Execute the chain and wait for the result
// 	finalMsg, err := runtime.ExecuteAndWait(context.Background(), "start_node", msg, nil)

// 	// 6. Assert the results
// 	assert.NoError(t, err)
// 	assert.NotNil(t, finalMsg)
// 	// The userCheck node doesn't modify the message, so it should be the same.
// 	assert.Equal(t, msg.Data(), finalMsg.Data())
// }
