package helper

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/neohet/matrix/pkg/registry"
	"gitlab.com/neohet/matrix/pkg/types"
)

// setupTestMsg creates a message with pre-populated dataT objects for testing.
func setupTestMsg(t *testing.T) types.RuleMsg {
	dataT := types.NewDataT()

	headersObj, err := dataT.NewItem("map_string_string", "headersObj")
	require.NoError(t, err)
	*(headersObj.Body().(*map[string]string)) = map[string]string{"X-Dynamic-Header": "dynamic-value"}

	bodyObj, err := dataT.NewItem("map_string_interface", "bodyObj")
	require.NoError(t, err)
	*(bodyObj.Body().(*map[string]interface{})) = map[string]interface{}{"user": "test", "id": 123}

	queryObj, err := dataT.NewItem("map_string_string", "queryObj")
	require.NoError(t, err)
	*(queryObj.Body().(*map[string]string)) = map[string]string{"q": "matrix", "limit": "10"}

	msg := types.NewMsg("TEST", `{"key":"value"}`, make(map[string]string), dataT)
	msg.Metadata()["requestId"] = "req-123"
	return msg
}

func TestMapRuleMsgToHttpRequest_NewMappings(t *testing.T) {
	ctx := registry.NewMinimalNodeCtx("test-node")
	msg := setupTestMsg(t)

	t.Run("Dynamic Headers and Body from dataT", func(t *testing.T) {
		cfg := HttpRequestMap{
			URL:    "http://test.com/api",
			Method: "POST",
			Headers: &HttpMappingSource{
				From: &DynamicTarget{Path: "dataT.headersObj"},
			},
			Body: &HttpMappingSource{
				From: &DynamicTarget{Path: "dataT.bodyObj"},
			},
		}

		req, err := MapRuleMsgToHttpRequest(ctx, msg, cfg, "5s")
		require.NoError(t, err)

		assert.Equal(t, "dynamic-value", req.Header.Get("X-Dynamic-Header"))
		bodyBytes, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		assert.JSONEq(t, `{"id":123,"user":"test"}`, string(bodyBytes))
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
	})

	t.Run("Mixed Dynamic and Static Body", func(t *testing.T) {
		cfg := HttpRequestMap{
			URL:    "http://test.com/api",
			Method: "POST",
			Body: &HttpMappingSource{
				From: &DynamicTarget{Path: "dataT.bodyObj"},
				Params: []HttpParam{
					{Name: "id", Mapping: HttpMapping{From: "'456'"}}, // Use single quotes for string literal
					{Name: "newField", Mapping: HttpMapping{From: "metadata.requestId"}},
				},
			},
		}

		req, err := MapRuleMsgToHttpRequest(ctx, msg, cfg, "5s")
		require.NoError(t, err)

		bodyBytes, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		// Note: JSON numbers are float64 by default. The literal '456' is a string.
		// For JSONEq to work correctly, we expect the final JSON to have "456" as a string.
		assert.JSONEq(t, `{"id":"456","user":"test", "newField": "req-123"}`, string(bodyBytes))
	})

	t.Run("Dynamic Query Params from dataT", func(t *testing.T) {
		cfg := HttpRequestMap{
			URL:    "http://test.com/api",
			Method: "GET",
			QueryParams: &HttpMappingSource{
				From: &DynamicTarget{Path: "dataT.queryObj"},
			},
		}

		req, err := MapRuleMsgToHttpRequest(ctx, msg, cfg, "5s")
		require.NoError(t, err)

		q := req.URL.Query()
		assert.Equal(t, "matrix", q.Get("q"))
		assert.Equal(t, "10", q.Get("limit"))
	})

	t.Run("Static Literal Value Mapping", func(t *testing.T) {
		cfg := HttpRequestMap{
			URL:    "http://test.com/api",
			Method: "GET",
			Headers: &HttpMappingSource{
				Params: []HttpParam{
					{Name: "Content-Type", Mapping: HttpMapping{From: "'application/json'"}},
					{Name: "X-Custom-Static", Mapping: HttpMapping{From: `"static-value"`}},
				},
			},
		}

		req, err := MapRuleMsgToHttpRequest(ctx, msg, cfg, "5s")
		require.NoError(t, err)

		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
		assert.Equal(t, "static-value", req.Header.Get("X-Custom-Static"))
	})

	t.Run("Body from msg.Data backward compatibility", func(t *testing.T) {
		msgWithData := types.NewMsg("TEST", `{"from":"data"}`, make(map[string]string), types.NewDataT())
		cfg := HttpRequestMap{
			URL:    "http://test.com/api",
			Method: "POST",
			Body: &HttpMappingSource{
				From: &DynamicTarget{Path: "data"},
			},
		}

		req, err := MapRuleMsgToHttpRequest(ctx, msgWithData, cfg, "5s")
		require.NoError(t, err)

		bodyBytes, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		assert.JSONEq(t, `{"from":"data"}`, string(bodyBytes))
	})

	t.Run("Empty and nil mappings", func(t *testing.T) {
		cfg := HttpRequestMap{
			URL:         "http://test.com/api",
			Method:      "GET",
			Headers:     nil,
			QueryParams: &HttpMappingSource{}, // Empty source
			Body:        nil,
		}

		req, err := MapRuleMsgToHttpRequest(ctx, msg, cfg, "5s")
		require.NoError(t, err)
		assert.Empty(t, req.Header)
		assert.Empty(t, req.URL.RawQuery)
		assert.Nil(t, req.Body)
	})
}

func TestMapHttpResponseToRuleMsg_NewMappings(t *testing.T) {
	ctx := registry.NewMinimalNodeCtx("test-node")

	t.Run("Map response body and headers to dataT", func(t *testing.T) {
		outMsg := setupTestMsg(t)
		respBody := `{"status":"ok","data":{"id":1,"name":"test-item"}}`
		resp := &http.Response{
			StatusCode: 200,
			Header: http.Header{
				"Content-Type":  []string{"application/json"},
				"X-Response-Id": []string{"resp-abc"},
			},
			Body: io.NopCloser(strings.NewReader(respBody)),
		}
		cfg := HttpResponseMap{
			StatusCodeTarget: "meta.httpStatus",
			Headers: &HttpMappingSource{
				From: &DynamicTarget{
					Path:      "dataT.responseHeaders",
					DefineSID: "map_string_string",
				},
				Params: []HttpParam{
					{Name: "X-Response-Id", Mapping: HttpMapping{To: "metadata.responseId"}},
				},
			},
			Body: &HttpMappingSource{
				From: &DynamicTarget{
					Path:      "dataT.responseBody",
					DefineSID: "map_string_interface",
				},
			},
		}

		endTime := time.Now()
		startTime := endTime.Add(-100 * time.Millisecond)
		err := MapHttpResponseToRuleMsg(ctx, resp, outMsg, cfg, startTime, endTime, nil)
		require.NoError(t, err)

		assert.Equal(t, "200", outMsg.Metadata()["meta.httpStatus"])
		assert.Equal(t, "resp-abc", outMsg.Metadata()["responseId"])

		headersObj, ok := outMsg.DataT().Get("responseHeaders")
		require.True(t, ok)
		headersMap := headersObj.Body().(*map[string]string)
		assert.Equal(t, "application/json", (*headersMap)["Content-Type"])

		bodyObj, ok := outMsg.DataT().Get("responseBody")
		require.True(t, ok, "responseBody object should be created in dataT")

		bodyMap, ok := bodyObj.Body().(*map[string]interface{})
		require.True(t, ok, "response body should be a map")
		assert.Equal(t, "ok", (*bodyMap)["status"])
	})

	t.Run("Map response with static body field extraction", func(t *testing.T) {
		outMsg := setupTestMsg(t)
		respBody := `{"status":"ok","data":{"id":1,"name":"test-item"}, "trace_id":"xyz-987"}`
		resp := &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(respBody)),
		}
		cfg := HttpResponseMap{
			Body: &HttpMappingSource{
				Params: []HttpParam{
					{Name: "status", Mapping: HttpMapping{To: "metadata.requestStatus"}},
					{Name: "trace_id", Mapping: HttpMapping{To: "metadata.traceId"}},
				},
			},
		}

		endTime := time.Now()
		startTime := endTime.Add(-50 * time.Millisecond)
		err := MapHttpResponseToRuleMsg(ctx, resp, outMsg, cfg, startTime, endTime, nil)
		require.NoError(t, err)

		assert.Equal(t, "ok", outMsg.Metadata()["requestStatus"])
		assert.Equal(t, "xyz-987", outMsg.Metadata()["traceId"])
	})

	t.Run("Map request error to metadata", func(t *testing.T) {
		outMsg := setupTestMsg(t)
		requestErr := errors.New("connection refused")
		cfg := HttpResponseMap{
			ErrorTarget: "meta.httpError",
		}

		// Note: resp is nil when there is a request error
		endTime := time.Now()
		startTime := endTime.Add(-200 * time.Millisecond)
		err := MapHttpResponseToRuleMsg(ctx, nil, outMsg, cfg, startTime, endTime, requestErr)
		require.NoError(t, err)

		assert.Equal(t, "connection refused", outMsg.Metadata()["meta.httpError"])
	})
}
