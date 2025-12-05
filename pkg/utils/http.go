package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
)

// ParseRequestBody parses the request body based on Content-Type.
func ParseRequestBody(r *http.Request) (map[string]any, error) {
	var bodyData map[string]any
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			return nil, fmt.Errorf("failed to parse multipart form: %w", err)
		}
		bodyData = make(map[string]any)
		for k, v := range r.MultipartForm.Value {
			if len(v) > 0 {
				bodyData[k] = v[0]
			}
		}
		for k, v := range r.MultipartForm.File {
			if len(v) > 0 {
				bodyData[k] = v[0]
			}
		}
	} else if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&bodyData); err != nil && err.Error() != "EOF" {
			return nil, fmt.Errorf("failed to decode request body: %w", err)
		}
		// Close body after reading
		r.Body.Close()
	}
	return bodyData, nil
}

// ParsePathParams extracts path parameters from the request context or URL.
func ParsePathParams(r *http.Request, configPath string) map[string]string {
	pathParams := ExtractPathParams(r.Context())
	// Fallback to manual extraction if context does not contain params.
	if pathParams == nil {
		pathParams = ManualExtractPathParams(configPath, r.URL.Path)
	}
	return pathParams
}

// ExtractPathParams extracts path parameters from the request context.
func ExtractPathParams(ctx context.Context) map[string]string {
	params, ok := ctx.Value(httprouter.ParamsKey).(httprouter.Params)
	if !ok {
		return nil
	}
	pathParams := make(map[string]string)
	for _, param := range params {
		pathParams[param.Key] = param.Value
	}
	return pathParams
}

// ManualExtractPathParams manually extracts path parameters by comparing the configured path pattern
// with the actual request path. This is a fallback for environments where httprouter.Params
// are not correctly passed through the context.
func ManualExtractPathParams(configPath, requestPath string) map[string]string {
	configParts := strings.Split(strings.Trim(configPath, "/"), "/")
	requestParts := strings.Split(strings.Trim(requestPath, "/"), "/")

	if len(configParts) != len(requestParts) {
		return nil
	}

	params := make(map[string]string)
	for i, part := range configParts {
		if strings.HasPrefix(part, ":") {
			key := strings.TrimPrefix(part, ":")
			params[key] = requestParts[i]
		}
	}
	return params
}
