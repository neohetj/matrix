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

package external

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/NeohetJ/Matrix/internal/registry"
	"github.com/NeohetJ/Matrix/pkg/message"
	"github.com/NeohetJ/Matrix/pkg/types"
	"github.com/NeohetJ/Matrix/pkg/utils"
	tutils "github.com/NeohetJ/Matrix/test/utils"
	"github.com/stretchr/testify/assert"
)

const (
	MapStringInterfaceSID = "MapStringInterfaceV1_0"
)

func init() {
	registry.Default.CoreObjRegistry.Register(
		message.NewCoreObjDef(&map[string]interface{}{}, MapStringInterfaceSID, "Generic map object"),
	)
}

// newNodeForTest 是一个辅助函数，用于为测试创建和初始化一个新的 HttpClientNode。
func newNodeForTest(t *testing.T, config HttpClientNodeConfiguration) *HttpClientNode {
	// We call New() to get a node with the default client initialized.
	node := httpClientNodePrototype.New().(*HttpClientNode)
	cfgMap, err := utils.ToMap(config)
	assert.NoError(t, err)
	err = node.Init(cfgMap)
	assert.NoError(t, err)
	return node
}

// 测试函数: TestHttpClientNode_Init
func TestHttpClientNode_Init(t *testing.T) {
	// 测试点: 确保当配置中的 `defineSid` 未注册时，节点初始化会失败。
	t.Run("should fail if defineSid is not registered", func(t *testing.T) {
		node := httpClientNodePrototype.New().(*HttpClientNode)
		config := HttpClientNodeConfiguration{
			Response: types.HttpResponseMap{
				Body: types.EndpointIOPacket{
					MapAll: utils.Ptr("rulemsg://data?sid=NonExistentV1_0"),
				},
			},
		}
		cfgMap, err := utils.ToMap(config)
		assert.NoError(t, err)
		err = node.Init(cfgMap)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SID 'NonExistentV1_0' is not registered")
	})
}

// mockHttpDoer is a mock client for testing network failures.
type mockHttpDoer struct {
	doFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHttpDoer) Do(req *http.Request) (*http.Response, error) {
	return m.doFunc(req)
}

// 测试函数: TestHttpClientNode_OnMsg
func TestHttpClientNode_OnMsg(t *testing.T) {
	// 测试点: 最简单的GET请求，验证客户端与测试服务器的基本连通性。
	t.Run("super simple GET request", func(t *testing.T) {
		config := HttpClientNodeConfiguration{
			Request: types.HttpRequestMap{
				URL:    "http://example.com/simple",
				Method: "GET",
			},
		}
		node := newNodeForTest(t, config)
		node.client = &mockHttpDoer{
			doFunc: func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "http://example.com/simple", req.URL.String())
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString("")),
				}, nil
			},
		}

		ctx := tutils.NewMockNodeCtx()
		node.OnMsg(ctx, tutils.NewTestRuleMsg())

		assert.Nil(t, ctx.FailureErr)
		assert.NotNil(t, ctx.SuccessMsg)
	})

	// 测试点: 验证指南示例1：使用POST方法发送JSON并接收JSON响应的成功路径。
	t.Run("Guide Example 1: POST JSON and receive JSON", func(t *testing.T) {
		config := HttpClientNodeConfiguration{
			Request: types.HttpRequestMap{
				URL:    "http://example.com/users/${dataT.userInfo.userId}",
				Method: "POST",
				Body:   types.EndpointIOPacket{MapAll: utils.Ptr("rulemsg://dataT/userInfo")},
			},
			Response: types.HttpResponseMap{
				StatusCodeTarget: "httpStatusCode",
				Body:             types.EndpointIOPacket{MapAll: utils.Ptr("rulemsg://dataT/apiResult?sid=" + MapStringInterfaceSID)},
			},
		}
		node := newNodeForTest(t, config)
		node.client = &mockHttpDoer{
			doFunc: func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "http://example.com/users/123", req.URL.String())
				bodyBytes, _ := io.ReadAll(req.Body)
				assert.JSONEq(t, `{"userId": 123, "userName": "Alice"}`, string(bodyBytes))

				respBody := io.NopCloser(bytes.NewBufferString(`{"status":"success"}`))
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       respBody,
					// 必须要设置
					Header: http.Header{"Content-Type": []string{"application/json"}},
				}, nil
			},
		}

		msg := tutils.NewTestRuleMsg()
		userInfo, _ := msg.DataT().NewItem(MapStringInterfaceSID, "userInfo")
		json.Unmarshal([]byte(`{"userId": 123, "userName": "Alice"}`), userInfo.Body())

		ctx := tutils.NewMockNodeCtx()
		node.OnMsg(ctx, msg)

		assert.Nil(t, ctx.FailureErr)
		assert.NotNil(t, ctx.SuccessMsg)
		apiResult, ok := ctx.SuccessMsg.DataT().Get("apiResult")
		assert.True(t, ok)
		assert.Equal(t, "success", (*apiResult.Body().(*map[string]interface{}))["status"])
	})

	// 测试点: 验证指南示例2：使用GET方法并正确映射查询参数的成功路径。
	t.Run("Guide Example 2: GET with QueryParams", func(t *testing.T) {
		config := HttpClientNodeConfiguration{
			Request: types.HttpRequestMap{
				URL:         "http://example.com/items",
				Method:      "GET",
				QueryParams: types.EndpointIOPacket{MapAll: utils.Ptr("rulemsg://dataT/queryParams")},
			},
		}
		node := newNodeForTest(t, config)
		node.client = &mockHttpDoer{
			doFunc: func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "http://example.com/items?page=1&pageSize=10", req.URL.String())
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString("")),
				}, nil
			},
		}
		msg := tutils.NewTestRuleMsg()
		queryParams, _ := msg.DataT().NewItem(MapStringInterfaceSID, "queryParams")
		json.Unmarshal([]byte(`{"page": 1, "pageSize": 10}`), queryParams.Body())

		ctx := tutils.NewMockNodeCtx()
		node.OnMsg(ctx, msg)
		assert.Nil(t, ctx.FailureErr)
	})

	// 测试点: 验证指南示例4：组合使用动态`from`和静态`params`来构建请求体的功能。
	t.Run("Guide Example 4: Combine request body", func(t *testing.T) {
		config := HttpClientNodeConfiguration{
			Request: types.HttpRequestMap{
				URL:    "http://example.com/complex",
				Method: "POST",
				Body: types.EndpointIOPacket{
					MapAll: utils.Ptr("rulemsg://dataT/baseInfo"),
					Fields: []types.EndpointIOField{
						{Name: "dynamic_field", BindPath: "rulemsg://metadata/dynamicId", Type: "string"},
						{Name: "static_field", BindPath: "'static_value'", Type: "string"},
					},
				},
			},
		}
		node := newNodeForTest(t, config)
		node.client = &mockHttpDoer{
			doFunc: func(req *http.Request) (*http.Response, error) {
				bodyBytes, _ := io.ReadAll(req.Body)
				assert.JSONEq(t, `{"common":"value", "dynamic_field":"xyz", "static_field":"static_value"}`, string(bodyBytes))
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(""))}, nil
			},
		}
		msg := tutils.NewTestRuleMsg()
		msg.Metadata()["dynamicId"] = "xyz"
		baseInfo, _ := msg.DataT().NewItem(MapStringInterfaceSID, "baseInfo")
		json.Unmarshal([]byte(`{"common": "value"}`), baseInfo.Body())

		ctx := tutils.NewMockNodeCtx()
		node.OnMsg(ctx, msg)
		assert.Nil(t, ctx.FailureErr)
	})

	// 测试点: 验证指南示例5：确保HttpResponseMap中定义的所有元信息目标键都能被正确填充。
	t.Run("Guide Example 5: Map all response metadata", func(t *testing.T) {
		config := HttpClientNodeConfiguration{
			Request: types.HttpRequestMap{URL: "http://example.com", Method: "GET"},
			Response: types.HttpResponseMap{
				StatusCodeTarget: "http.status_code",
				LatencyMsTarget:  "http.latency_ms",
				ErrorTarget:      "http.error",
			},
		}
		node := newNodeForTest(t, config)
		node.client = &mockHttpDoer{
			doFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusCreated, Body: io.NopCloser(bytes.NewBufferString(""))}, nil
			},
		}

		ctx := tutils.NewMockNodeCtx()
		node.OnMsg(ctx, tutils.NewTestRuleMsg())

		assert.Nil(t, ctx.FailureErr)
		meta := ctx.SuccessMsg.Metadata()
		assert.Equal(t, "201", meta["http.status_code"])
		assert.NotEmpty(t, meta["http.latency_ms"])
	})

	// 测试点: 确保在发生网络连接错误时，节点能正确地走向失败链路，并报告相应的错误信息。
	t.Run("should fail on connection error and map error metadata", func(t *testing.T) {
		config := HttpClientNodeConfiguration{
			Request:  types.HttpRequestMap{URL: "http://localhost:9999", Method: "GET"},
			Response: types.HttpResponseMap{ErrorTarget: "http.error"},
		}
		node := newNodeForTest(t, config)
		// Inject a mock client that always returns an error.
		node.client = &mockHttpDoer{
			doFunc: func(req *http.Request) (*http.Response, error) {
				return nil, fmt.Errorf("simulated connection refused")
			},
		}

		ctx := tutils.NewMockNodeCtx()
		node.OnMsg(ctx, tutils.NewTestRuleMsg())

		assert.NotNil(t, ctx.FailureErr)
		assert.Equal(t, FaultHttpSendFailed.Message, tutils.GetRootError(ctx.FailureErr).Message)
		assert.Contains(t, ctx.FailureMsg.Metadata()["http.error"], "simulated connection refused")
	})
}
