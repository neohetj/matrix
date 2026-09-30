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
	"net/http/httptest"
	"testing"

	"github.com/neohetj/matrix/internal/contract"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/pkg/utils"
	tutils "github.com/neohetj/matrix/test/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Messages used by these tests exercise the real DataT and CoreObj implementations.
func newHTTPMessage() types.RuleMsg {
	return contract.NewDefaultRuleMsg("http-test", "", nil, contract.NewDataT())
}

func addHTTPMap(t *testing.T, msg types.RuleMsg, id, body string) {
	t.Helper()
	obj, err := msg.DataT().NewItem(cnst.SID_MAP_STRING_INTERFACE, id)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal([]byte(body), obj.Body()))
}

// newNodeForTest 是一个辅助函数，用于为测试创建和初始化一个新的 HttpClientNode。
func newNodeForTest(t *testing.T, config HttpClientNodeConfiguration) *HttpClientNode {
	t.Helper()
	// We call New() to get a node with the default client initialized.
	node := httpClientNodePrototype.New().(*HttpClientNode)
	cfgMap, err := utils.ToMap(config)
	require.NoError(t, err)
	err = node.Init(cfgMap)
	require.NoError(t, err)
	return node
}

// 测试函数: TestHttpClientNode_Init
func TestHttpClientNode_Init(t *testing.T) {
	// 测试点: 确保当配置中的 `response SID` 未注册时，节点初始化会失败。
	t.Run("should fail if response SID is not registered", func(t *testing.T) {
		node := httpClientNodePrototype.New().(*HttpClientNode)
		config := HttpClientNodeConfiguration{
			Response: types.HttpResponseMap{
				Body: types.EndpointIOPacket{
					MapAll: utils.Ptr("rulemsg://dataT/result?sid=NonExistent_V1"),
				},
			},
		}
		cfgMap, err := utils.ToMap(config)
		require.NoError(t, err)
		err = node.Init(cfgMap)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SID 'NonExistent_V1' is not registered")
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
		node.OnMsg(ctx, newHTTPMessage())

		require.NoError(t, ctx.FailureErr)
		require.NotNil(t, ctx.SuccessMsg)
	})

	// 测试点: 验证指南示例1：使用POST方法发送JSON并接收JSON响应的成功路径。
	t.Run("Guide Example 1: POST JSON and receive JSON", func(t *testing.T) {
		type capturedRequest struct {
			method, path, contentType, body string
			err                             error
		}
		requests := make(chan capturedRequest, 1)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			requests <- capturedRequest{r.Method, r.URL.RequestURI(), r.Header.Get("Content-Type"), string(body), err}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"status":"success"}`)
		}))
		t.Cleanup(server.Close)
		config := HttpClientNodeConfiguration{
			Request: types.HttpRequestMap{
				URL:    server.URL + "/users/${rulemsg://dataT/userInfo.userId?sid=MapStringInterface}",
				Method: "POST",
				Body:   types.EndpointIOPacket{MapAll: utils.Ptr("rulemsg://dataT/userInfo?sid=" + cnst.SID_MAP_STRING_INTERFACE)},
			},
			Response: types.HttpResponseMap{
				StatusCodeTarget: "httpStatusCode",
				Body:             types.EndpointIOPacket{MapAll: utils.Ptr("rulemsg://dataT/apiResult?sid=" + cnst.SID_MAP_STRING_INTERFACE)},
			},
		}
		node := newNodeForTest(t, config)

		msg := newHTTPMessage()
		addHTTPMap(t, msg, "userInfo", `{"userId": 123, "userName": "Alice"}`)

		ctx := tutils.NewMockNodeCtx()
		node.OnMsg(ctx, msg)

		require.NoError(t, ctx.FailureErr)
		require.NotNil(t, ctx.SuccessMsg)
		apiResult, ok := ctx.SuccessMsg.DataT().Get("apiResult")
		require.True(t, ok)
		assert.Equal(t, "success", (*apiResult.Body().(*map[string]interface{}))["status"])
		assert.Equal(t, "201", ctx.SuccessMsg.Metadata()["httpStatusCode"])
		assert.NotContains(t, msg.Metadata(), "httpStatusCode")
		select {
		case request := <-requests:
			require.NoError(t, request.err)
			assert.Equal(t, "POST", request.method)
			assert.Equal(t, "/users/123", request.path)
			assert.Equal(t, "application/json", request.contentType)
			assert.JSONEq(t, `{"userId":123,"userName":"Alice"}`, request.body)
		default:
			t.Fatal("no HTTP request reached the test server")
		}
	})

	// 测试点: 验证指南示例2：使用GET方法并正确映射查询参数的成功路径。
	t.Run("Guide Example 2: GET with QueryParams", func(t *testing.T) {
		config := HttpClientNodeConfiguration{
			Request: types.HttpRequestMap{
				URL:         "http://example.com/items",
				Method:      "GET",
				QueryParams: types.EndpointIOPacket{MapAll: utils.Ptr("rulemsg://dataT/queryParams?sid=" + cnst.SID_MAP_STRING_INTERFACE)},
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
		msg := newHTTPMessage()
		addHTTPMap(t, msg, "queryParams", `{"page": 1, "pageSize": 10}`)

		ctx := tutils.NewMockNodeCtx()
		node.OnMsg(ctx, msg)
		require.NoError(t, ctx.FailureErr)
	})

	// 测试点: 验证指南示例4：组合使用`mapAll`和`fields`来构建请求体的功能。
	t.Run("Guide Example 4: Combine request body", func(t *testing.T) {
		config := HttpClientNodeConfiguration{
			Request: types.HttpRequestMap{
				URL:    "http://example.com/complex",
				Method: "POST",
				Body: types.EndpointIOPacket{
					MapAll: utils.Ptr("rulemsg://dataT/baseInfo?sid=" + cnst.SID_MAP_STRING_INTERFACE),
					Fields: []types.EndpointIOField{
						{Name: "dynamic_field", BindPath: "rulemsg://metadata/dynamicId", Type: "string"},
						{Name: "static_field", DefaultValue: "static_value", Type: "string"},
					},
				},
			},
		}
		node := newNodeForTest(t, config)
		node.client = &mockHttpDoer{
			doFunc: func(req *http.Request) (*http.Response, error) {
				bodyBytes, err := io.ReadAll(req.Body)
				require.NoError(t, err)
				assert.JSONEq(t, `{"common":"value", "dynamic_field":"xyz", "static_field":"static_value"}`, string(bodyBytes))
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(""))}, nil
			},
		}
		msg := newHTTPMessage()
		msg.Metadata()["dynamicId"] = "xyz"
		addHTTPMap(t, msg, "baseInfo", `{"common": "value"}`)

		ctx := tutils.NewMockNodeCtx()
		node.OnMsg(ctx, msg)
		require.NoError(t, ctx.FailureErr)
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
		node.OnMsg(ctx, newHTTPMessage())

		require.NoError(t, ctx.FailureErr)
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
		node.OnMsg(ctx, newHTTPMessage())

		require.Error(t, ctx.FailureErr)
		assert.Equal(t, FaultHttpSendFailed.Message, tutils.GetRootError(ctx.FailureErr).Message)
		assert.Contains(t, ctx.FailureMsg.Metadata()["http.error"], "simulated connection refused")
	})
}
