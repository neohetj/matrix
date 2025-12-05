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

package helper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/NeohetJ/Matrix/pkg/cnst"
	"github.com/NeohetJ/Matrix/pkg/message"
	"github.com/NeohetJ/Matrix/pkg/trace"
	"github.com/NeohetJ/Matrix/pkg/types"
	"github.com/NeohetJ/Matrix/pkg/utils"
)

const (
	DefaultLatencyMetaKey = "httpLatencyMs"
	DefaultErrorMetaKey   = "httpError"
)

var (
	RequestDecodingFailed = &types.Fault{Code: cnst.CodeRequestDecodingFailed, Message: "failed to decode request body"}
	MapAllDataFailed      = &types.Fault{Code: cnst.CodeInvalidMappingFormat, Message: "failed to map all data"}
	RequiredFieldMissing  = &types.Fault{Code: cnst.CodeRequiredFieldMissing, Message: "required field is missing"}
	FieldConversionFailed = &types.Fault{Code: cnst.CodeFieldConversionFailed, Message: "failed to convert field"}
	SetFieldFailed        = &types.Fault{Code: cnst.CodeInvalidMappingFormat, Message: "failed to set field"}
	ExtractMapAllFailed   = &types.Fault{Code: cnst.CodeInvalidMappingFormat, Message: "failed to extract MapAll"}
	ExpectedFileType      = &types.Fault{Code: cnst.CodeFieldConversionFailed, Message: "expected file type"}
)

// HeaderProvider implements ValueProvider for http.Header.
type HeaderProvider http.Header

func (p HeaderProvider) GetValue(name string) (any, bool, error) {
	v := http.Header(p).Get(name)
	return v, v != "", nil
}

func (p HeaderProvider) GetAll() (any, bool, error) {
	m := make(map[string]string)
	for k, v := range p {
		m[k] = strings.Join(v, ", ")
	}
	return m, true, nil
}

// QueryProvider implements ValueProvider for url.Values (query parameters).
type QueryProvider url.Values

func (p QueryProvider) GetValue(name string) (any, bool, error) {
	q := url.Values(p)
	// Check if the field name itself indicates an array (e.g., "ids[]")
	if strings.HasSuffix(name, "[]") {
		vals, ok := q[name]
		if ok && len(vals) > 0 {
			return vals, true, nil
		}
		return nil, false, nil
	}

	val := q.Get(name)
	if val != "" {
		return val, true, nil
	}

	vals, ok := q[name]
	if ok && len(vals) > 0 {
		if len(vals) == 1 {
			return vals[0], true, nil
		}
		return vals, true, nil
	}
	return nil, false, nil
}

func (p QueryProvider) GetAll() (any, bool, error) {
	return url.Values(p), true, nil
}

// BodyProvider implements ValueProvider for structural body data.
type BodyProvider struct {
	Data any
}

func (p BodyProvider) GetValue(name string) (any, bool, error) {
	val, found, err := utils.ExtractByPath(p.Data, name)
	return val, found, err
}

func (p BodyProvider) GetAll() (any, bool, error) {
	return p.Data, true, nil
}

// -----------------------------------------------------------------------------
// HTTP Client Helpers
// -----------------------------------------------------------------------------

// MapRuleMsgToHttpRequest builds an *http.Request using ProcessOutbound logic.
func MapRuleMsgToHttpRequest(ctx types.NodeCtx, msg types.RuleMsg, cfg types.HttpRequestMap, defaultTimeout string) (*http.Request, context.CancelFunc, error) {
	// 1. URL & Method substitution
	targetURL, err := message.ReplaceRuleMsg(cfg.URL, msg)
	if err != nil {
		return nil, nil, types.InvalidParams.Wrap(fmt.Errorf("URL placeholder resolution failed: %w", err))
	}
	targetMethod, err := message.ReplaceRuleMsg(cfg.Method, msg)
	if err != nil {
		return nil, nil, types.InvalidParams.Wrap(fmt.Errorf("Method placeholder resolution failed: %w", err))
	}

	// 2. Build Body
	bodyReader, contentType, err := buildClientRequestBody(ctx, msg, cfg.Body)
	if err != nil {
		return nil, nil, types.InvalidParams.Wrap(fmt.Errorf("body build failed: %w", err))
	}

	// 3. Create Request
	timeout, err := time.ParseDuration(defaultTimeout)
	if err != nil || timeout <= 0 {
		timeout = 60 * time.Second
	}
	reqContext, cancel := context.WithTimeout(ctx.GetContext(), timeout)

	httpReq, err := http.NewRequestWithContext(reqContext, strings.ToUpper(targetMethod), targetURL, bodyReader)
	if err != nil {
		cancel()
		return nil, nil, types.InternalError.Wrap(fmt.Errorf("request creation failed: %w", err))
	}

	if contentType != "" {
		httpReq.Header.Set("Content-Type", contentType)
	}

	// 4. Headers
	headersMap, err := ProcessOutbound(ctx, msg, cfg.Headers, RuleMsgProvider{Msg: msg})
	if err != nil {
		cancel()
		return nil, nil, types.InvalidParams.Wrap(fmt.Errorf("header processing failed: %w", err))
	}
	for k, v := range headersMap {
		if k != "" { // Skip special keys if any
			httpReq.Header.Set(k, fmt.Sprintf("%v", v))
		}
	}

	// 5. Query Params
	queryMap, err := ProcessOutbound(ctx, msg, cfg.QueryParams, RuleMsgProvider{Msg: msg})
	if err != nil {
		cancel()
		return nil, nil, types.InvalidParams.Wrap(fmt.Errorf("query param processing failed: %w", err))
	}
	q := httpReq.URL.Query()
	for k, v := range queryMap {
		if k != "" {
			// Handle slice values for query params (e.g., ids[]=1&ids[]=2)
			// But ProcessOutbound usually returns scalar or basic types.
			// If v is a slice, we should iterate.
			// For now simple string conversion:
			q.Add(k, fmt.Sprintf("%v", v))
		}
	}
	httpReq.URL.RawQuery = q.Encode()

	// 6. Trace Propagation
	if cfg.PropagateMeta {
		metaToPropagate := trace.GetMetadataToPropagate(msg.Metadata(), cfg.PropagateKeys)
		for k, v := range metaToPropagate {
			httpReq.Header.Set(k, v)
		}
	}

	return httpReq, cancel, nil
}

func buildClientRequestBody(ctx types.NodeCtx, msg types.RuleMsg, packet types.EndpointIOPacket) (io.Reader, string, error) {
	// Optimization: If packet is empty, return nil
	if (packet.MapAll == nil || *packet.MapAll == "") && len(packet.Fields) == 0 {
		if len(msg.Data()) == 0 {
			return nil, "", nil
		}
		// Use msg.Data if no body mapping is defined and format is suitable
		if msg.DataFormat() == cnst.TEXT || msg.DataFormat() == cnst.JSON {
			contentType := "text/plain"
			if msg.DataFormat() == cnst.JSON {
				contentType = "application/json"
			}
			return strings.NewReader(string(msg.Data())), contentType, nil
		}
		// If IMAGE/BYTES, handle appropriately if needed, or default to nil
		if msg.DataFormat() == cnst.IMAGE || msg.DataFormat() == cnst.BYTES {
			return strings.NewReader(string(msg.Data())), "application/octet-stream", nil
		}
		return nil, "", nil
	}

	dataMap, err := ProcessOutbound(ctx, msg, packet, RuleMsgProvider{Msg: msg})
	if err != nil {
		return nil, "", err
	}

	// Check if we have a "raw" body from MapAll
	if raw, ok := dataMap[""]; ok && len(packet.Fields) == 0 {
		// Single raw value mapping
		strVal := fmt.Sprintf("%v", raw)
		// Try to guess content type or default to text/plain?
		// For backward compatibility with "data" mapping
		contentType := "text/plain"
		if msg.DataFormat() == cnst.JSON {
			contentType = "application/json"
		}
		return strings.NewReader(strVal), contentType, nil
	}

	// Otherwise treat as JSON object
	// dataMap contains all fields constructed by ProcessOutbound (including nested ones via dot notation)
	bytes, err := json.Marshal(dataMap)
	if err != nil {
		return nil, "", types.InternalError.Wrap(fmt.Errorf("json marshal failed: %w", err))
	}
	return strings.NewReader(string(bytes)), "application/json", nil
}

// MapHttpResponseToRuleMsg maps an *http.Response TO a RuleMsg using ProcessInbound logic.
func MapHttpResponseToRuleMsg(ctx types.NodeCtx, resp *http.Response, msg types.RuleMsg, cfg types.HttpResponseMap, startTime, endTime time.Time, requestErr error) error {
	// 1. Metadata (Latency, Status, Error)
	latency := endTime.Sub(startTime)
	msg.Metadata()[getTargetKey(cfg.LatencyMsTarget, DefaultLatencyMetaKey)] = fmt.Sprintf("%d", latency.Milliseconds())
	if cfg.StartTimeMsTarget != "" {
		msg.Metadata()[cfg.StartTimeMsTarget] = fmt.Sprintf("%d", startTime.UnixMilli())
	}
	if cfg.EndTimeMsTarget != "" {
		msg.Metadata()[cfg.EndTimeMsTarget] = fmt.Sprintf("%d", endTime.UnixMilli())
	}
	if requestErr != nil {
		msg.Metadata()[getTargetKey(cfg.ErrorTarget, DefaultErrorMetaKey)] = requestErr.Error()
		return nil
	}

	if cfg.StatusCodeTarget != "" {
		msg.Metadata()[cfg.StatusCodeTarget] = fmt.Sprintf("%d", resp.StatusCode)
	}

	// 2. Map Headers
	if err := ProcessInbound(ctx, msg, cfg.Headers, HeaderProvider(resp.Header)); err != nil {
		ctx.Warn("Failed to map response headers", "error", err)
	}

	// 3. Map Body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.InternalError.Wrap(fmt.Errorf("read body failed: %w", err))
	}
	// Always set raw data first
	contentType := resp.Header.Get("Content-Type")
	var format cnst.MFormat
	if strings.Contains(contentType, "application/json") {
		format = cnst.JSON
	} else if strings.Contains(contentType, "image/") {
		format = cnst.IMAGE
	} else {
		format = cnst.TEXT
	}
	msg.SetData(string(bodyBytes), format)

	// If JSON, allow structural mapping
	if msg.DataFormat() == cnst.JSON && len(bodyBytes) > 0 {
		var bodyData any
		if err := json.Unmarshal(bodyBytes, &bodyData); err == nil {
			if err := ProcessInbound(ctx, msg, cfg.Body, BodyProvider{Data: bodyData}); err != nil {
				ctx.Warn("Failed to map response body", "error", err)
			}
		}
	}

	return nil
}

// -----------------------------------------------------------------------------
// HTTP Server Helpers
// -----------------------------------------------------------------------------

// MapHttpRequestToRuleMsg maps an HTTP request (parts) TO a RuleMsg using ProcessInbound logic.
// Used by: HttpEndpoint.
func MapHttpRequestToRuleMsg(ctx types.NodeCtx, msg types.RuleMsg, reqDef types.HttpRequestDef, r *http.Request, configPath string) error {
	// Parse body first
	bodyData, err := utils.ParseRequestBody(r)
	if err != nil {
		if strings.Contains(err.Error(), "decode request body") {
			return RequestDecodingFailed.Wrap(err)
		}
		return types.InvalidParams.Wrap(err)
	}

	// Parse path params
	pathParams := utils.ParsePathParams(r, configPath)

	// 1. Path Params (Fields only)
	pathPacket := types.EndpointIOPacket{Fields: reqDef.PathParams}
	// Convert map[string]string to map[string]any for MapProvider
	pathMap := make(map[string]any)
	for k, v := range pathParams {
		pathMap[k] = v
	}
	if err := ProcessInbound(ctx, msg, pathPacket, MapProvider(pathMap)); err != nil {
		return err
	}

	// 2. Query Params
	if err := ProcessInbound(ctx, msg, reqDef.QueryParams, QueryProvider(r.URL.Query())); err != nil {
		return err
	}

	// 3. Headers
	if err := ProcessInbound(ctx, msg, reqDef.Headers, HeaderProvider(r.Header)); err != nil {
		return err
	}

	// 4. Body
	if err := ProcessInbound(ctx, msg, reqDef.Body, BodyProvider{Data: bodyData}); err != nil {
		return err
	}

	return nil
}

// MapRuleMsgToHttpResponse maps a RuleMsg TO HTTP response parts (body, headers).
// Used by: HttpEndpoint.
func MapRuleMsgToHttpResponse(ctx types.NodeCtx, msg types.RuleMsg, respDef types.HttpResponseDef) (map[string]any, map[string]string, int, error) {
	// Set default status code
	statusCode := respDef.SuccessCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	if msg == nil || msg.DataT() == nil {
		return make(map[string]any), make(map[string]string), statusCode, nil
	}

	// 1. Process Body
	body, err := ProcessOutbound(ctx, msg, respDef.Body, RuleMsgProvider{Msg: msg})
	if err != nil {
		return nil, nil, 0, err // ProcessOutbound now returns *types.Fault
	}
	// Check for "MapAll" raw result override
	if raw, ok := body[""]; ok && len(respDef.Body.Fields) == 0 {
		if m, ok := raw.(map[string]any); ok {
			body = m
		} else {
			delete(body, "")
		}
	}

	// 2. Process Headers
	headersRaw, err := ProcessOutbound(ctx, msg, respDef.Headers, RuleMsgProvider{Msg: msg})
	if err != nil {
		return nil, nil, 0, err // ProcessOutbound now returns *types.Fault
	}
	headers := make(map[string]string)
	for k, v := range headersRaw {
		if k != "" {
			headers[k] = fmt.Sprintf("%v", v)
		} else if m, ok := v.(map[string]any); ok {
			for hk, hv := range m {
				headers[hk] = fmt.Sprintf("%v", hv)
			}
		}
	}

	return body, headers, statusCode, nil
}

func getTargetKey(configKey, defaultKey string) string {
	if configKey != "" {
		return configKey
	}
	return defaultKey
}
