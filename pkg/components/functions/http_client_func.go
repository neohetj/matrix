package functions

// import (
// 	"bytes"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"net/http"
// 	"strings"
// 	"time"

// 	"gitlab.com/neohet/matrix/pkg/registry"
// 	"gitlab.com/neohet/matrix/pkg/trace"
// 	"gitlab.com/neohet/matrix/pkg/types"
// )

// // --- Constants Definition ---
// const (
// 	// NodeID is the unique identifier for this function node.
// 	HttpClientFuncID = "httpClient"

// 	// CoreObj Type System Identifiers (SID).
// 	HttpRequestV1_0_SID  = "HttpRequestV1_0"
// 	HttpResponseV1_0_SID = "HttpResponseV1_0"

// 	// Input/Output parameter names.
// 	ParamNameHttpRequest  = "httpRequest"
// 	ParamNameHttpResponse = "httpResponse"

// 	// Configuration keys.
// 	ConfigKeyTimeout       = "timeout"
// 	ConfigKeyPropagateMeta = "propagateMeta"
// 	ConfigKeyPropagateKeys = "propagateKeys"
// )

// // --- Error Definitions ---
// var (
// 	ErrHttpRequestFailed  = &types.ErrorObj{Code: 202502001, Message: "HTTP request creation failed"}
// 	ErrHttpSendFailed     = &types.ErrorObj{Code: 202502002, Message: "HTTP request sending failed"}
// 	ErrHttpReadBodyFailed = &types.ErrorObj{Code: 202502003, Message: "Failed to read HTTP response body"}
// )

// // --- CoreObj Definitions ---

// // HttpRequest defines the structure for the HTTP request input object.
// type HttpRequest struct {
// 	URL     string            `json:"url"`
// 	Method  string            `json:"method"`
// 	Headers map[string]string `json:"headers"`
// 	Body    any               `json:"body"`
// }

// // HttpResponse defines the structure for the HTTP response output object.
// type HttpResponse struct {
// 	StatusCode int         `json:"statusCode"`
// 	Headers    http.Header `json:"headers"`
// 	Body       any         `json:"body"`
// }

// func init() {
// 	// Register CoreObj definitions.
// 	registry.Default.CoreObjRegistry.Register(
// 		types.NewCoreObjDef(&HttpRequest{}, HttpRequestV1_0_SID, "HTTP请求对象"),
// 		types.NewCoreObjDef(&HttpResponse{}, HttpResponseV1_0_SID, "HTTP响应对象"),
// 	)

// 	// Register the function node.
// 	registry.Default.NodeFuncManager.Register(&types.NodeFuncObject{
// 		Func: HttpClientFunc,
// 		FuncObject: types.FuncObject{
// 			ID:        HttpClientFuncID,
// 			Name:      "HTTP Client",
// 			Desc:      "Sends an HTTP request based on an input object.",
// 			Dimension: "External",
// 			Tags:      []string{"http", "rest", "api"},
// 			Version:   "1.0.0",
// 			Configuration: types.FuncObjConfiguration{
// 				Name:     HttpClientFuncID,
// 				FuncDesc: "Receives an HttpRequest object, sends the request, and outputs an HttpResponse object.",
// 				Business: []types.DynamicConfigField{
// 					{ID: ConfigKeyTimeout, Name: "Timeout", Desc: "Request timeout duration.", Default: "30s", Required: true, Type: "duration"},
// 					{ID: ConfigKeyPropagateMeta, Name: "Propagate Metadata", Desc: "If true, propagates specified metadata keys as request headers.", Default: false, Type: "bool"},
// 					{ID: ConfigKeyPropagateKeys, Name: "Metadata Keys to Propagate", Desc: "List of metadata keys to propagate. If empty, only ExecutionID is propagated. Use ['*'] for all.", Default: []string{}, Type: "[]string"},
// 				},
// 				Inputs: []types.IOObject{
// 					{ParamName: ParamNameHttpRequest, DefineSID: HttpRequestV1_0_SID, Desc: "The HTTP request details object.", Required: true},
// 				},
// 				Outputs: []types.IOObject{
// 					{ParamName: ParamNameHttpResponse, DefineSID: HttpResponseV1_0_SID, Desc: "The HTTP response details object."},
// 				},
// 				Errors: []*types.ErrorObj{
// 					ErrHttpRequestFailed,
// 					ErrHttpSendFailed,
// 					ErrHttpReadBodyFailed,
// 				},
// 			},
// 		},
// 	})
// }

// // HttpClientFunc sends an HTTP request based on a structured input object.
// func HttpClientFunc(ctx types.NodeCtx, msg types.RuleMsg) {
// 	// 1. Get input object from DataT
// 	reqUntyped, err := msg.DataT().GetByParam(ctx, ParamNameHttpRequest)
// 	if err != nil {
// 		ctx.TellFailure(msg, types.DefInvalidParams.Wrap(fmt.Errorf("input object '%s' not found", ParamNameHttpRequest)))
// 		return
// 	}
// 	reqData, ok := reqUntyped.Body().(*HttpRequest)
// 	if !ok {
// 		ctx.TellFailure(msg, types.DefInvalidParams.Wrap(fmt.Errorf("failed to cast input to *HttpRequest")))
// 		return
// 	}

// 	// 2. Get configuration
// 	config := ctx.Config()
// 	timeoutStr, _ := config[ConfigKeyTimeout].(string)
// 	timeout, err := time.ParseDuration(timeoutStr)
// 	if err != nil {
// 		timeout = 30 * time.Second // Fallback to default
// 	}
// 	propagateMeta, _ := config[ConfigKeyPropagateMeta].(bool)
// 	var propagateKeys []string
// 	if keys, ok := config[ConfigKeyPropagateKeys].([]interface{}); ok {
// 		for _, k := range keys {
// 			if keyStr, ok := k.(string); ok {
// 				propagateKeys = append(propagateKeys, keyStr)
// 			}
// 		}
// 	}

// 	// 3. Prepare request body
// 	var bodyReader io.Reader
// 	if reqData.Body != nil {
// 		bodyBytes, err := json.Marshal(reqData.Body)
// 		if err != nil {
// 			ctx.TellFailure(msg, ErrHttpRequestFailed.Wrap(fmt.Errorf("failed to marshal request body: %w", err)))
// 			return
// 		}
// 		bodyReader = bytes.NewBuffer(bodyBytes)
// 	}

// 	// 4. Create HTTP request
// 	req, err := http.NewRequestWithContext(ctx.GetContext(), strings.ToUpper(reqData.Method), reqData.URL, bodyReader)
// 	if err != nil {
// 		ctx.TellFailure(msg, ErrHttpRequestFailed.Wrap(err))
// 		return
// 	}

// 	// 5. Set headers
// 	for k, v := range reqData.Headers {
// 		req.Header.Set(k, v)
// 	}
// 	if reqData.Body != nil && req.Header.Get("Content-Type") == "" {
// 		req.Header.Set("Content-Type", "application/json")
// 	}
// 	if propagateMeta {
// 		metaToPropagate := trace.GetMetadataToPropagate(msg.Metadata(), propagateKeys)
// 		for k, v := range metaToPropagate {
// 			req.Header.Set(k, v)
// 		}
// 	}

// 	// 6. Send request
// 	client := &http.Client{Timeout: timeout}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		ctx.TellFailure(msg, ErrHttpSendFailed.Wrap(err))
// 		return
// 	}
// 	defer resp.Body.Close()

// 	// 7. Read response body
// 	respBodyBytes, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		ctx.TellFailure(msg, ErrHttpReadBodyFailed.Wrap(err))
// 		return
// 	}
// 	var respBodyData any
// 	if err := json.Unmarshal(respBodyBytes, &respBodyData); err != nil {
// 		// If not a valid JSON, treat as a raw string.
// 		respBodyData = string(respBodyBytes)
// 	}

// 	// 8. Create and populate output object
// 	respObj, err := msg.DataT().NewItemByParam(ctx, ParamNameHttpResponse)
// 	if err != nil {
// 		ctx.TellFailure(msg, types.DefInternalError.Wrap(fmt.Errorf("failed to create output object: %w", err)))
// 		return
// 	}
// 	httpResponse := respObj.Body().(*HttpResponse)
// 	httpResponse.StatusCode = resp.StatusCode
// 	httpResponse.Headers = resp.Header
// 	httpResponse.Body = respBodyData

// 	// 9. Notify success
// 	ctx.TellSuccess(msg)
// }
