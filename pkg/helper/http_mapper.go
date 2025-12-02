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
	"strings"
	"time"

	"gitlab.com/neohet/matrix/pkg/trace"
	"gitlab.com/neohet/matrix/pkg/types"
	"gitlab.com/neohet/matrix/pkg/utils"
)

// HttpParam defines a mapping from a source to a target.
// This is a shared structure used by http_endpoint and http_client.
type HttpParam struct {
	Name     string         `json:"name"`
	Type     types.DataType `json:"type"`
	Required bool           `json:"required"`
	Mapping  HttpMapping    `json:"mapping"`
}

// HttpMapping specifies the 'from' and 'to' paths for data transformation.
type HttpMapping struct {
	From      string `json:"from"` // e.g., "metadata.token", "dataT.userObj.id", "data.field"
	To        string `json:"to"`   // e.g., "header.X-Auth-Token", "body.userId"
	DefineSID string `json:"defineSid,omitempty"`
}

// DynamicTarget specifies a target path for dynamic mapping and the type to create if it doesn't exist.
type DynamicTarget struct {
	Path      string `json:"path"`                // e.g., "dataT.responseHeaders"
	DefineSID string `json:"defineSid,omitempty"` // e.g., "map_string_string". Optional, but required if the target dataT object might not exist.
}

// HttpMappingSource provides a unified way to define mappings,
// supporting both a dynamic source map and a list of static parameters.
// Both can be used simultaneously.
type HttpMappingSource struct {
	// From specifies a dynamic source to be mapped to a target path.
	From *DynamicTarget `json:"from,omitempty"`
	// Params specifies a list of static, item-by-item mappings.
	Params []HttpParam `json:"params,omitempty"`
}

// HttpRequestMap defines how to construct an HTTP request from a RuleMsg.
type HttpRequestMap struct {
	URL    string `json:"url"`
	Method string `json:"method"`

	// Headers defines all header mappings, both dynamic and static.
	Headers *HttpMappingSource `json:"headers,omitempty"`

	// QueryParams defines all query parameter mappings.
	QueryParams *HttpMappingSource `json:"queryParams,omitempty"`

	// Body defines the request body mappings.
	Body *HttpMappingSource `json:"body,omitempty"`

	PropagateMeta bool     `json:"propagateMeta"`
	PropagateKeys []string `json:"propagateKeys"`
}

// HttpResponseMap defines how to map an HTTP response back to a RuleMsg.
type HttpResponseMap struct {
	// StatusCodeTarget (可选) 指定一个Metadata键，用于存储HTTP响应的状态码。
	StatusCodeTarget string `json:"statusCodeTarget,omitempty"`
	// LatencyMsTarget (可选) 指定一个Metadata键，用于存储从请求发送到接收到响应的延迟（毫秒）。
	LatencyMsTarget string `json:"latencyMsTarget,omitempty"`
	// ErrorTarget (可选) 指定一个Metadata键，用于存储请求过程中发生的网络错误或连接错误。
	ErrorTarget string `json:"errorTarget,omitempty"`
	// StartTimeMsTarget (可选) 指定一个Metadata键，用于存储请求开始时间的Unix毫秒时间戳。
	StartTimeMsTarget string `json:"startTimeMsTarget,omitempty"`
	// EndTimeMsTarget (可选) 指定一个Metadata键，用于存储请求结束时间的Unix毫秒时间戳。
	EndTimeMsTarget string `json:"endTimeMsTarget,omitempty"`

	// Headers defines how to map response headers back to the RuleMsg.
	Headers *HttpMappingSource `json:"headers,omitempty"`

	// Body defines how to map the response body back to the RuleMsg.
	Body *HttpMappingSource `json:"body,omitempty"`
}

// MapRuleMsgToHttpRequest builds an *http.Request from a types.RuleMsg based on a mapping configuration.
func MapRuleMsgToHttpRequest(ctx types.NodeCtx, msg types.RuleMsg, cfg HttpRequestMap, defaultTimeout string) (*http.Request, error) {
	// 1. Build data source for placeholder replacement
	dataSource := BuildDataSource(msg)
	targetURL := utils.ReplacePlaceholders(cfg.URL, dataSource)
	targetMethod := utils.ReplacePlaceholders(cfg.Method, dataSource)

	// 2. Build Request Body and Content-Type Header
	bodyReader, contentType, err := buildRequestBody(ctx, msg, cfg.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to build request body: %w", err)
	}

	// 3. Create Request
	timeout, _ := time.ParseDuration(defaultTimeout)
	// The context will be managed by the http.Client, so we don't call cancel() here.
	reqContext, _ := context.WithTimeout(ctx.GetContext(), timeout)

	httpReq, err := http.NewRequestWithContext(reqContext, strings.ToUpper(targetMethod), targetURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	// 4. Set Content-Type if derived from body mapping
	if contentType != "" {
		httpReq.Header.Set("Content-Type", contentType)
	}

	// 5. Set Headers
	if err := applyHeaders(ctx, msg, cfg.Headers, httpReq); err != nil {
		return nil, fmt.Errorf("failed to apply headers: %w", err)
	}

	// 6. Set Query Params
	if err := applyQueryParams(ctx, msg, cfg.QueryParams, httpReq); err != nil {
		return nil, fmt.Errorf("failed to apply query params: %w", err)
	}

	// 7. Propagate Trace Metadata
	if cfg.PropagateMeta {
		metaToPropagate := trace.GetMetadataToPropagate(msg.Metadata(), cfg.PropagateKeys)
		for k, v := range metaToPropagate {
			httpReq.Header.Set(k, v)
		}
	}

	return httpReq, nil
}

func applyHeaders(ctx types.NodeCtx, msg types.RuleMsg, source *HttpMappingSource, req *http.Request) error {
	if source == nil {
		return nil
	}

	// Process dynamic source first
	if source.From != nil && source.From.Path != "" {
		val, found, err := ExtractFromMsgByPath(msg, source.From.Path)
		if err != nil {
			return fmt.Errorf("error extracting headers from %s: %w", source.From.Path, err)
		}
		if found {
			headerMap, err := utils.ToMap(val)
			if err != nil {
				return fmt.Errorf("source %s is not a map-like structure for headers: %w", source.From.Path, err)
			}
			for k, v := range headerMap {
				req.Header.Set(k, fmt.Sprintf("%v", v))
			}
		}
	}

	// Process static params, potentially overriding dynamic values
	for _, param := range source.Params {
		val, found, err := ExtractFromMsgByPath(msg, param.Mapping.From)
		if err != nil {
			ctx.Warn("Failed to extract header value", "from", param.Mapping.From, "error", err)
			continue
		}
		if found {
			req.Header.Set(param.Name, fmt.Sprintf("%v", val))
		}
	}
	return nil
}

func applyQueryParams(ctx types.NodeCtx, msg types.RuleMsg, source *HttpMappingSource, req *http.Request) error {
	if source == nil {
		return nil
	}
	q := req.URL.Query()

	// Process dynamic source first
	if source.From != nil && source.From.Path != "" {
		val, found, err := ExtractFromMsgByPath(msg, source.From.Path)
		if err != nil {
			return fmt.Errorf("error extracting query params from %s: %w", source.From.Path, err)
		}
		if found {
			paramMap, err := utils.ToMap(val)
			if err != nil {
				return fmt.Errorf("source %s is not a map-like structure for query params: %w", source.From.Path, err)
			}
			for k, v := range paramMap {
				q.Add(k, fmt.Sprintf("%v", v))
			}
		}
	}

	// Process static params
	for _, param := range source.Params {
		val, found, err := ExtractFromMsgByPath(msg, param.Mapping.From)
		if err != nil {
			ctx.Warn("Failed to extract query param value", "from", param.Mapping.From, "error", err)
			continue
		}
		if found {
			q.Add(param.Name, fmt.Sprintf("%v", val))
		}
	}

	req.URL.RawQuery = q.Encode()
	return nil
}

func buildRequestBody(ctx types.NodeCtx, msg types.RuleMsg, source *HttpMappingSource) (io.Reader, string, error) {
	if source == nil {
		return nil, "", nil
	}

	bodyMap := make(map[string]interface{})

	// Process dynamic source first
	if source.From != nil && source.From.Path != "" {
		// Special case for "data", to maintain backward compatibility
		if source.From.Path == "data" {
			var contentType string
			switch msg.DataFormat() {
			case types.JSON:
				contentType = "application/json"
			case types.TEXT:
				contentType = "text/plain"
			case types.BYTES:
				contentType = "application/octet-stream"
			default:
				contentType = "text/plain"
			}
			return strings.NewReader(msg.Data()), contentType, nil
		}

		val, found, err := ExtractFromMsgByPath(msg, source.From.Path)
		if err != nil {
			return nil, "", fmt.Errorf("error extracting body from %s: %w", source.From.Path, err)
		}
		if found {
			fromMap, err := utils.ToMap(val)
			if err != nil {
				return nil, "", fmt.Errorf("source %s is not a map-like structure for body: %w", source.From.Path, err)
			}
			for k, v := range fromMap {
				bodyMap[k] = v
			}
		}
	}

	// Process static params, adding to or overriding the dynamic map
	for _, param := range source.Params {
		val, found, err := ExtractFromMsgByPath(msg, param.Mapping.From)
		if err != nil {
			return nil, "", fmt.Errorf("error extracting body field %s: %w", param.Mapping.From, err)
		}
		if !found {
			if param.Required {
				return nil, "", fmt.Errorf("required body field '%s' not found", param.Mapping.From)
			}
			continue
		}
		bodyMap[param.Name] = val
	}

	if len(bodyMap) == 0 {
		return nil, "", nil
	}

	bodyBytes, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal request body: %w", err)
	}
	return strings.NewReader(string(bodyBytes)), "application/json", nil
}

const (
	DefaultLatencyMetaKey = "httpLatencyMs"
	DefaultErrorMetaKey   = "httpError"
)

// getTargetKey returns the configured key if not empty, otherwise returns the default key.
func getTargetKey(configKey, defaultKey string) string {
	if configKey != "" {
		return configKey
	}
	return defaultKey
}

// MapHttpResponseToRuleMsg maps an *http.Response to a types.RuleMsg based on a mapping configuration.
func MapHttpResponseToRuleMsg(ctx types.NodeCtx, resp *http.Response, msg types.RuleMsg, cfg HttpResponseMap, startTime, endTime time.Time, requestErr error) error {
	// 1. Map performance metrics and request error
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
		return nil // If there was a request error, there's no response to map.
	}

	// 2. Read the raw response body
	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// 3. Set raw body and data format on the message
	msg.SetData(string(respBodyBytes))
	contentType := resp.Header.Get("Content-Type")
	switch {
	case strings.Contains(contentType, "application/json"):
		msg.WithDataFormat(types.JSON)
	case strings.Contains(contentType, "text/"):
		msg.WithDataFormat(types.TEXT)
	default:
		msg.WithDataFormat(types.BYTES)
	}

	// 4. Map status code
	if cfg.StatusCodeTarget != "" {
		msg.Metadata()[cfg.StatusCodeTarget] = fmt.Sprintf("%d", resp.StatusCode)
	}

	// 5. Map response headers
	if cfg.Headers != nil {
		// Dynamic mapping: dump all headers to a target map
		if cfg.Headers.From != nil && cfg.Headers.From.Path != "" {
			headerMap := make(map[string]string)
			for k, v := range resp.Header {
				headerMap[k] = strings.Join(v, ", ")
			}
			if err := setInMsgByPath(ctx, msg, cfg.Headers.From.Path, headerMap, cfg.Headers.From.DefineSID); err != nil {
				ctx.Warn("Failed to set all headers in message", "to", cfg.Headers.From.Path, "error", err)
			}
		}
		// Static mapping: map individual headers
		for _, param := range cfg.Headers.Params {
			headerValue := resp.Header.Get(param.Name)
			if headerValue != "" {
				if err := setInMsgByPath(ctx, msg, param.Mapping.To, headerValue, param.Mapping.DefineSID); err != nil {
					ctx.Warn("Failed to set header in message", "to", param.Mapping.To, "error", err)
				}
			}
		}
	}

	// 6. Map response body
	if cfg.Body != nil && len(respBodyBytes) > 0 && msg.DataFormat() == types.JSON {
		// Dynamic mapping: unmarshal entire body to a target object
		if cfg.Body.From != nil && cfg.Body.From.Path != "" {
			var targetObj interface{}
			if err := json.Unmarshal(respBodyBytes, &targetObj); err != nil {
				ctx.Warn("Failed to unmarshal response body", "error", err)
			} else {
				if err := setInMsgByPath(ctx, msg, cfg.Body.From.Path, targetObj, cfg.Body.From.DefineSID); err != nil {
					ctx.Warn("Failed to set response body in message", "to", cfg.Body.From.Path, "error", err)
				}
			}
		}
		// Static mapping: extract individual fields from body
		if len(cfg.Body.Params) > 0 {
			var bodyData map[string]interface{}
			if err := json.Unmarshal(respBodyBytes, &bodyData); err != nil {
				return fmt.Errorf("failed to unmarshal json response body for field mapping: %w", err)
			}
			for _, param := range cfg.Body.Params {
				val, found, err := utils.ExtractByPath(bodyData, param.Name)
				if err != nil {
					ctx.Warn("Failed to extract field from response body", "field", param.Name, "error", err)
					continue
				}
				if found {
					if err := setInMsgByPath(ctx, msg, param.Mapping.To, val, param.Mapping.DefineSID); err != nil {
						ctx.Warn("Failed to set body field in message", "to", param.Mapping.To, "error", err)
					}
				}
			}
		}
	}

	return nil
}

// setInMsgByPath is a helper to set a value in a RuleMsg using a dot-path.
func setInMsgByPath(ctx types.NodeCtx, msg types.RuleMsg, path string, value any, defineSid string) error {
	parts := strings.SplitN(path, ".", 2)
	if len(parts) < 2 {
		return fmt.Errorf("invalid target path format: %s", path)
	}
	msgType, msgKey := parts[0], parts[1]

	switch msgType {
	case "metadata":
		msg.Metadata()[msgKey] = fmt.Sprintf("%v", value)
	case "dataT":
		objParts := strings.SplitN(msgKey, ".", 2)
		objID := objParts[0]

		obj, found := msg.DataT().Get(objID)
		if !found {
			if defineSid == "" {
				return fmt.Errorf("dataT object with id '%s' not found and no defineSid provided", objID)
			}
			var err error
			obj, err = msg.DataT().NewItem(defineSid, objID)
			if err != nil {
				return fmt.Errorf("failed to create new dataT item with sid '%s': %w", defineSid, err)
			}
		}

		// Case 1: Path is like "dataT.myObj", set the entire object body.
		if len(objParts) == 1 {
			// Handle headers, which are map[string]string
			if headerMap, ok := value.(map[string]string); ok {
				if target, ok := obj.Body().(*map[string]string); ok {
					*target = headerMap
					return nil
				}
			}

			// Handle body, which is typically map[string]interface{} after json.Unmarshal
			if bodyMap, ok := value.(map[string]interface{}); ok {
				if err := utils.Decode(bodyMap, obj.Body()); err != nil {
					return fmt.Errorf("failed to decode value into dataT object '%s' body: %w", objID, err)
				}
				return nil
			}
			return fmt.Errorf("unsupported type for whole object assignment: %T", value)
		}

		// Case 2: Path is like "dataT.myObj.field", set a field within the object.
		fieldPath := objParts[1]
		objMap, err := utils.ToMap(obj.Body())
		if err != nil {
			return fmt.Errorf("failed to convert dataT object body to map for setting value: %w", err)
		}

		setValueByDotPath(objMap, fieldPath, value)

		if err := utils.Decode(objMap, obj.Body()); err != nil {
			return fmt.Errorf("failed to decode map back to dataT object body: %w", err)
		}
	default:
		return fmt.Errorf("unsupported message type in target path: %s", msgType)
	}
	return nil
}

// setValueByDotPath is a simplified helper to set a value in a nested map.
func setValueByDotPath(data map[string]interface{}, path string, value interface{}) {
	parts := strings.Split(path, ".")
	current := data
	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
		} else {
			if _, ok := current[part].(map[string]interface{}); !ok {
				current[part] = make(map[string]interface{})
			}
			current = current[part].(map[string]interface{})
		}
	}
}
