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
	"os"
	"strings"
	"testing"

	"github.com/neohetj/matrix/internal/contract"
	matrixlog "github.com/neohetj/matrix/internal/log"
	"github.com/neohetj/matrix/test/utils"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// Set a simple logger for tests to see warnings.
	matrixlog.SetLogger(&utils.TestLogger{})
	os.Exit(m.Run())
}

func TestDefaultCoreObjRegistry_Register(t *testing.T) {
	t.Run("should register valid defs", func(t *testing.T) {
		registry := NewCoreObjRegistry()
		def1 := contract.NewDefaultCoreObjDef(&struct{}{}, "TestObjectV1_0", "desc")
		registry.Register(def1)

		retDef, ok := registry.Get("TestObjectV1_0")
		assert.True(t, ok)
		assert.Equal(t, def1, retDef)
	})

	t.Run("should log warning for non-compliant SID", func(t *testing.T) {
		// Setup mock logger
		mockLog := &utils.MockLogger{}
		originalLogger := matrixlog.GetLogger()
		matrixlog.SetLogger(mockLog)
		defer matrixlog.SetLogger(originalLogger)

		registry := NewCoreObjRegistry()
		def1 := contract.NewDefaultCoreObjDef(&struct{}{}, "invalidSid", "desc")
		registry.Register(def1)

		// Check if it was still registered
		_, ok := registry.Get("invalidSid")
		assert.True(t, ok, "object should still be registered despite non-compliant SID")

		// Check for warning log
		logOutput := mockLog.String()
		assert.True(t, strings.Contains(logOutput, "does not conform to the recommended format"), "should log a warning for non-compliant SID")
		assert.True(t, strings.Contains(logOutput, "invalidSid"), "warning log should contain the invalid SID")
	})

	t.Run("should not log warning for compliant SID", func(t *testing.T) {
		mockLog := &utils.MockLogger{}
		originalLogger := matrixlog.GetLogger()
		matrixlog.SetLogger(mockLog)
		defer matrixlog.SetLogger(originalLogger)

		registry := NewCoreObjRegistry()
		def1 := contract.NewDefaultCoreObjDef(&struct{}{}, "GoodSidV1_0", "desc")
		registry.Register(def1)

		logOutput := mockLog.String()
		assert.Empty(t, logOutput, "should not log any warning for compliant SID")
	})
}
