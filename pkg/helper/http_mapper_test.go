package helper

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertFaultCode(t *testing.T, err error, expectedCode cnst.ErrCode) {
	var fault *types.Fault
	if errors.As(err, &fault) {
		assert.Equal(t, expectedCode, fault.Code, "Expected fault code %d, but got %d", expectedCode, fault.Code)
	} else {
		assert.Fail(t, "Expected error to be of type *types.Fault")
	}
}

// setupTestMsg creates a message with pre-populated dataT objects for testing.
func setupTestMsg(t *testing.T) types.RuleMsg {
	dataT := types.NewDataT()

	headersObj, err := dataT.NewItem(cnst.SID_MAP_STRING_STRING, "headersObj")
	require.NoError(t, err)
	*(headersObj.Body().(*map[string]string)) = map[string]string{"X-Dynamic-Header": "dynamic-value"}

	bodyObj, err := dataT.NewItem(cnst.SID_MAP_STRING_INTERFACE, "bodyObj")
	require.NoError(t, err)
	*(bodyObj.Body().(*map[string]interface{})) = map[string]interface{}{"user": "test", "id": 123}

	queryObj, err := dataT.NewItem(cnst.SID_MAP_STRING_STRING, "queryObj")
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
		cfg := types.HttpRequestMap{
			URL:    "http://test.com/api",
			Method: "POST",
			Headers: types.EndpointIOPacket{
				MapAll: utils.Ptr(fmt.Sprintf("rulemsg://dataT/headersObj?sid=%s", cnst.SID_MAP_STRING_STRING)),
			},
			Body: types.EndpointIOPacket{
				MapAll: utils.Ptr(fmt.Sprintf("rulemsg://dataT/bodyObj?sid=%s", cnst.SID_MAP_STRING_STRING)),
			},
		}

		req, cancel, err := MapRuleMsgToHttpRequest(ctx, msg, cfg, "5s")
		require.NoError(t, err)
		defer cancel()

		assert.Equal(t, "dynamic-value", req.Header.Get("X-Dynamic-Header"))
		bodyBytes, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		assert.JSONEq(t, `{"id":123,"user":"test"}`, string(bodyBytes))
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
	})

	t.Run("Mixed Dynamic and Static Body", func(t *testing.T) {
		cfg := types.HttpRequestMap{
			URL:    "http://test.com/api",
			Method: "POST",
			Body: types.EndpointIOPacket{
				MapAll: utils.Ptr(fmt.Sprintf("rulemsg://dataT/bodyObj?sid=%s", cnst.SID_MAP_STRING_INTERFACE)),
				Fields: []types.EndpointIOField{
					{Name: "id", DefaultValue: "456", Type: "string"},
					{Name: "newField", BindPath: "rulemsg://metadata/requestId", Type: "string"},
				},
			},
		}

		req, cancel, err := MapRuleMsgToHttpRequest(ctx, msg, cfg, "5s")
		require.NoError(t, err)
		defer cancel()

		bodyBytes, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		// Note: JSON numbers are float64 by default. The literal '456' is a string.
		// For JSONEq to work correctly, we expect the final JSON to have "456" as a string.
		assert.JSONEq(t, `{"id":"456","user":"test", "newField": "req-123"}`, string(bodyBytes))
	})

	t.Run("Dynamic Query Params from dataT", func(t *testing.T) {
		cfg := types.HttpRequestMap{
			URL:    "http://test.com/api",
			Method: "GET",
			QueryParams: types.EndpointIOPacket{
				MapAll: utils.Ptr(fmt.Sprintf("rulemsg://dataT/queryObj?sid=%s", cnst.SID_MAP_STRING_STRING)),
			},
		}

		req, cancel, err := MapRuleMsgToHttpRequest(ctx, msg, cfg, "5s")
		require.NoError(t, err)
		defer cancel()

		q := req.URL.Query()
		assert.Equal(t, "matrix", q.Get("q"))
		assert.Equal(t, "10", q.Get("limit"))
	})

	t.Run("Static Literal Value Mapping", func(t *testing.T) {
		cfg := types.HttpRequestMap{
			URL:    "http://test.com/api",
			Method: "GET",
			Headers: types.EndpointIOPacket{
				Fields: []types.EndpointIOField{
					// For static literal values, BindPath should be empty, and DefaultValue should be set.
					{Name: "Content-Type", DefaultValue: "application/json"},
					{Name: "X-Custom-Static", DefaultValue: "static-value"},
				},
			},
		}

		req, cancel, err := MapRuleMsgToHttpRequest(ctx, msg, cfg, "5s")
		require.NoError(t, err)
		defer cancel()

		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
		assert.Equal(t, "static-value", req.Header.Get("X-Custom-Static"))
	})

	t.Run("Body from msg.Data backward compatibility", func(t *testing.T) {
		msgWithData := types.NewMsg(string(cnst.TEXT), `{"from":"data"}`, make(map[string]string), types.NewDataT())
		cfg := types.HttpRequestMap{
			URL:    "http://test.com/api",
			Method: "POST",
			Body: types.EndpointIOPacket{
				MapAll: utils.Ptr(fmt.Sprintf("rulemsg://data?format=%s", cnst.TEXT)),
			},
		}

		req, cancel, err := MapRuleMsgToHttpRequest(ctx, msgWithData, cfg, "5s")
		require.NoError(t, err)
		defer cancel()

		bodyBytes, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		assert.JSONEq(t, `{"from":"data"}`, string(bodyBytes))
	})

	t.Run("Empty and nil mappings", func(t *testing.T) {
		emptyMsg := types.NewMsg("TEST", "", nil, types.NewDataT())
		cfg := types.HttpRequestMap{
			URL:         "http://test.com/api",
			Method:      "GET",
			Headers:     types.EndpointIOPacket{},
			QueryParams: types.EndpointIOPacket{}, // Empty source
			Body:        types.EndpointIOPacket{},
		}

		req, cancel, err := MapRuleMsgToHttpRequest(ctx, emptyMsg, cfg, "5s")
		require.NoError(t, err)
		defer cancel()
		assert.Empty(t, req.Header)
		assert.Empty(t, req.URL.RawQuery)
		assert.Nil(t, req.Body)
	})
}

type testValueProvider struct {
	val any
}

func (p *testValueProvider) GetValue(name string) (any, bool, error) {
	if p.val == nil {
		return nil, false, nil
	}
	if name == "bad_int" {
		return "not-an-int", true, nil
	}
	return p.val, true, nil
}

func (p *testValueProvider) GetAll() (any, bool, error) {
	return p.val, p.val != nil, nil
}

func TestProcessInbound_Errors(t *testing.T) {
	ctx := registry.NewMinimalNodeCtx("test-node")
	msg := types.NewMsg("TEST", "", nil, types.NewDataT())

	t.Run("Required Field Missing", func(t *testing.T) {
		packet := types.EndpointIOPacket{
			Fields: []types.EndpointIOField{
				{Name: "missing_field", Required: true},
			},
		}
		provider := &testValueProvider{val: nil}
		err := ProcessInbound(ctx, msg, packet, provider)
		assert.Error(t, err)
		assertFaultCode(t, err, cnst.CodeRequiredFieldMissing)
	})

	t.Run("Field Conversion Failed", func(t *testing.T) {
		packet := types.EndpointIOPacket{
			Fields: []types.EndpointIOField{
				{Name: "bad_int", Type: cnst.INT, Required: true},
			},
		}
		provider := &testValueProvider{val: "some-val"}
		err := ProcessInbound(ctx, msg, packet, provider)
		assert.Error(t, err)
		assertFaultCode(t, err, cnst.CodeFieldConversionFailed)
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
		cfg := types.HttpResponseMap{
			StatusCodeTarget: "meta.httpStatus",
			Headers: types.EndpointIOPacket{
				MapAll: utils.Ptr(fmt.Sprintf("rulemsg://dataT/responseHeaders?sid=%s", cnst.SID_MAP_STRING_STRING)),
				Fields: []types.EndpointIOField{
					{Name: "X-Response-Id", BindPath: "rulemsg://metadata/responseId"}, // Client: Name=Source, BindPath=Target
				},
			},
			Body: types.EndpointIOPacket{
				MapAll: utils.Ptr(fmt.Sprintf("rulemsg://dataT/responseBody?sid=%s", cnst.SID_MAP_STRING_INTERFACE)),
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
		cfg := types.HttpResponseMap{
			Body: types.EndpointIOPacket{
				Fields: []types.EndpointIOField{
					{Name: "status", BindPath: "rulemsg://metadata/requestStatus"},
					{Name: "trace_id", BindPath: "rulemsg://metadata/traceId"},
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
		cfg := types.HttpResponseMap{
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
