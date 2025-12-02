package endpoint

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
	"gitlab.com/neohet/matrix/pkg/helper"
	"gitlab.com/neohet/matrix/pkg/registry"
	"gitlab.com/neohet/matrix/pkg/types"
	"gitlab.com/neohet/matrix/pkg/utils"
)

const (
	HttpEndpointNodeType = "endpoint/http"
)

var (
	ErrRequestDecodingFailed   = &types.ErrorObj{Code: int32(types.CodeRequestDecodingFailed), Message: "failed to decode request body"}
	ErrRequiredFieldMissing    = &types.ErrorObj{Code: int32(types.CodeRequiredFieldMissing), Message: "required field not found in http request"}
	ErrFieldConversionFailed   = &types.ErrorObj{Code: int32(types.CodeFieldConversionFailed), Message: "failed to convert field"}
	ErrInvalidMappingFormat    = &types.ErrorObj{Code: int32(types.CodeInvalidMappingFormat), Message: "invalid mapping format"}
	ErrDataTItemCreationFailed = &types.ErrorObj{Code: int32(types.CodeDataTItemCreationFailed), Message: "failed to create new DataT item"}
)

// HttpEndpoint is a specific type of PassiveEndpoint for handling HTTP requests.
type HttpEndpoint interface {
	types.PassiveEndpoint
	// HandleHttpRequest handles the incoming HTTP request.
	// The implementation should write the response to the http.ResponseWriter.
	// It should return an error if any issue occurs that needs to be handled by the adapter.
	HandleHttpRequest(w http.ResponseWriter, r *http.Request, executionID string, finalizer types.SnapshotFinalizer) (types.ChainError, error)
	GetHttpPath() string
	GetHttpMethod() string
	Configuration() HttpEndpointNodeConfiguration
}

// ErrorConverter defines a contract for converting Matrix's internal errors
// into application-specific error formats.
type ErrorConverter interface {
	Convert(ep HttpEndpoint, chainErr types.ChainError, originalErr error) error
}

// httpEndpointNodePrototype is the shared prototype instance used for registration.
var httpEndpointNodePrototype = &HttpEndpointNode{
	BaseNode: *types.NewBaseNode(HttpEndpointNodeType, types.NodeDefinition{
		Name:        "HTTP Endpoint V2",
		Description: "Receives HTTP requests and triggers a rule chain based on a unified definition.",
		Dimension:   "Endpoint",
		Tags:        []string{"endpoint", "http", "rest", "v2"},
		Version:     "2.0.0",
	}),
}

// Self-registering to the NodeManager
func init() {
	registry.Default.NodeManager.Register(httpEndpointNodePrototype)
}

// HttpEndpointNodeConfiguration holds the V2 configuration for the HttpEndpointNode.
type HttpEndpointNodeConfiguration struct {
	RuleChainID        string                      `json:"ruleChainId"`
	StartNodeID        string                      `json:"startNodeId,omitempty"`
	HttpMethod         string                      `json:"httpMethod"`
	HttpPath           string                      `json:"httpPath"`
	Description        string                      `json:"description"`
	EndpointDefinition types.EndpointDefinitionObj `json:"endpointDefinition"`
}

// HttpEndpointNode is a component that acts as an entry point for HTTP requests.
type HttpEndpointNode struct {
	types.BaseNode
	types.Instance
	nodeConfig  HttpEndpointNodeConfiguration
	runtimePool types.RuntimePool
}

// New creates a new instance of the node.
func (n *HttpEndpointNode) New() types.Node {
	return &HttpEndpointNode{BaseNode: n.BaseNode}
}

// Init initializes the node with its static configuration.
func (n *HttpEndpointNode) Init(config types.Config) error {
	if err := utils.Decode(config, &n.nodeConfig); err != nil {
		return types.ErrInvalidConfiguration.Wrap(fmt.Errorf("failed to decode http endpoint v2 config: %w", err))
	}
	if n.nodeConfig.RuleChainID == "" {
		return types.ErrInvalidConfiguration.Wrap(errors.New("config 'ruleChainId' is required"))
	}
	if n.nodeConfig.HttpMethod == "" || n.nodeConfig.HttpPath == "" {
		return types.ErrInvalidConfiguration.Wrap(errors.New("config 'httpMethod' and 'httpPath' are required"))
	}
	return nil
}

// SetRuntimePool implements the types.Endpoint interface.
func (n *HttpEndpointNode) SetRuntimePool(pool any) error {
	if p, ok := pool.(types.RuntimePool); ok {
		n.runtimePool = p
		return nil
	}
	return types.ErrInvalidConfiguration.Wrap(errors.New("provided pool is not of type types.RuntimePool"))
}

// GetHttpPath returns the configured HTTP path for routing.
func (n *HttpEndpointNode) GetHttpPath() string {
	return n.nodeConfig.HttpPath
}

// GetHttpMethod returns the configured HTTP method for routing.
func (n *HttpEndpointNode) GetHttpMethod() string {
	return n.nodeConfig.HttpMethod
}

// GetInstance implements the types.SharedNode interface, returning the node itself.
func (n *HttpEndpointNode) GetInstance() (any, error) {
	return n, nil
}

// Configuration returns the node's configuration for inspection.
func (n *HttpEndpointNode) Configuration() HttpEndpointNodeConfiguration {
	return n.nodeConfig
}

// ErrorResponse is the standard JSON structure for error responses.
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// writeJsonError is a helper to write a standard JSON error response.
func writeJsonError(w http.ResponseWriter, code int, message string, details error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	response := ErrorResponse{
		Code:    code,
		Message: message,
	}
	if details != nil {
		response.Details = details.Error()
	}
	json.NewEncoder(w).Encode(response)
}

// HandleHttpRequest is the core method that processes the incoming HTTP request.
func (n *HttpEndpointNode) HandleHttpRequest(w http.ResponseWriter, r *http.Request, executionID string, finalizer types.SnapshotFinalizer) (types.ChainError, error) {
	initialMsg, err := n.convertRequestToRuleMsg(r)
	if err != nil {
		return nil, types.ErrInvalidParams.Wrap(err)
	}

	if executionID != "" {
		initialMsg.Metadata()[types.ExecutionIDKey] = executionID
	}

	var rt types.Runtime
	var ok bool
	if n.runtimePool != nil {
		rt, ok = n.runtimePool.Get(n.nodeConfig.RuleChainID)
	} else {
		rt, ok = registry.Default.RuntimePool.Get(n.nodeConfig.RuleChainID)
	}

	if !ok {
		return nil, types.ErrInternal.Wrap(fmt.Errorf("runtime not found for rule chain: %s", n.nodeConfig.RuleChainID))
	}

	onEnd := func(msg types.RuleMsg, err error) {
		if executionID != "" && finalizer != nil {
			finalizer.FinalizeSnapshot(executionID)
		}
	}

	finalMsg, execErr := rt.ExecuteAndWait(r.Context(), n.nodeConfig.StartNodeID, initialMsg, onEnd)

	if execErr != nil {
		var matrixErr *types.ErrorObj
		if errors.As(execErr, &matrixErr) {
			return nil, matrixErr
		}
		return nil, types.ErrInternal.Wrap(execErr)
	}

	if finalMsg != nil {
		if errStr, ok := finalMsg.Metadata()[types.MetaError]; ok {
			chainErr := make(types.ChainError)
			chainErr[types.MetaError] = errStr

			if val, ok := finalMsg.Metadata()[types.MetaErrorNodeID]; ok {
				chainErr[types.MetaErrorNodeID] = val
			}
			if val, ok := finalMsg.Metadata()[types.MetaErrorNodeName]; ok {
				chainErr[types.MetaErrorNodeName] = val
			}
			if val, ok := finalMsg.Metadata()[types.MetaErrorTimestamp]; ok {
				chainErr[types.MetaErrorTimestamp] = val
			}
			if val, ok := finalMsg.Metadata()[types.MetaErrorCode]; ok {
				chainErr[types.MetaErrorCode] = val
			}

			// The original error is now the string from metadata, not from execErr.
			// The adapter layer will decide if it needs to be parsed further.
			return chainErr, nil
		}
	}

	responseBody, responseHeaders, statusCode, err := n.convertResponse(finalMsg)
	if err != nil {
		return nil, types.ErrInternal.Wrap(err)
	}

	for k, v := range responseHeaders {
		w.Header().Set(k, v)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(responseBody)

	return nil, nil
}

// manualExtractPathParams manually extracts path parameters by comparing the configured path pattern
// with the actual request path. This is a fallback for environments where httprouter.Params
// are not correctly passed through the context.
func manualExtractPathParams(configPath, requestPath string) map[string]string {
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

// extractPathParams extracts path parameters from the request context.
func extractPathParams(ctx context.Context) map[string]string {
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

// convertRequestToRuleMsg handles the conversion from an http.Request to a types.RuleMsg.
func (n *HttpEndpointNode) convertRequestToRuleMsg(r *http.Request) (types.RuleMsg, error) {
	// 1. Prepare data sources
	var bodyData map[string]any
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&bodyData); err != nil && err.Error() != "EOF" {
			return nil, ErrRequestDecodingFailed.Wrap(err)
		}
		defer r.Body.Close()
	}
	queryParams := r.URL.Query()
	headerParams := r.Header
	pathParams := extractPathParams(r.Context())
	// Fallback to manual extraction if context does not contain params.
	if pathParams == nil {
		pathParams = manualExtractPathParams(n.nodeConfig.HttpPath, r.URL.Path)
	}

	// 2. Create the initial message
	msg := types.NewMsg(n.nodeConfig.RuleChainID, "", make(types.Metadata), nil).WithDataFormat(types.JSON)
	pendingDataT := make(map[string]map[string]any)
	objDefineSids := make(map[string]string)

	// 3. Define a generic mapping processor
	processMapping := func(param types.HttpParam, valueProvider func(string) (any, bool)) error {
		rawValue, found := valueProvider(param.Name)
		if !found {
			if param.Required {
				return ErrRequiredFieldMissing.Wrap(fmt.Errorf("field: %s", param.Name))
			}
			return nil
		}

		convertedValue, err := utils.Convert(rawValue, param.Type)
		if err != nil {
			return ErrFieldConversionFailed.Wrap(fmt.Errorf("field: %s, error: %w", param.Name, err))
		}

		msgParts := strings.SplitN(param.Mapping.To, ".", 2)
		if len(msgParts) < 2 {
			return ErrInvalidMappingFormat.Wrap(fmt.Errorf("param: %s, format: %s", param.Name, param.Mapping.To))
		}
		msgType := msgParts[0]
		msgKey := msgParts[1]

		switch msgType {
		case "metadata":
			msg.Metadata()[msgKey] = fmt.Sprintf("%v", convertedValue)
		case "dataT":
			objParts := strings.SplitN(msgKey, ".", 2)
			if len(objParts) < 2 {
				return ErrInvalidMappingFormat.Wrap(fmt.Errorf("param: %s, format: %s", param.Name, msgKey))
			}
			objID, fieldPath := objParts[0], objParts[1]

			if _, ok := pendingDataT[objID]; !ok {
				pendingDataT[objID] = make(map[string]any)
			}
			setValueByDotPath(pendingDataT[objID], fieldPath, convertedValue)

			if _, ok := objDefineSids[objID]; !ok && param.Mapping.DefineSID != "" {
				objDefineSids[objID] = param.Mapping.DefineSID
			}
		}
		return nil
	}

	// 4. Process all parameter types
	reqDef := n.nodeConfig.EndpointDefinition.Request
	for _, p := range reqDef.PathParams {
		if err := processMapping(p, func(name string) (any, bool) { v, ok := pathParams[name]; return v, ok }); err != nil {
			return nil, err
		}
	}
	for _, p := range reqDef.QueryParams {
		// Check if the parameter type is defined as a slice/array.
		if strings.HasSuffix(p.Type, "[]") || strings.HasPrefix(p.Type, "[]") {
			// If it's a slice type, pass the whole slice.
			// The key for query arrays might be `ids[]` or just `ids`.
			if values, ok := queryParams[p.Name]; ok && len(values) > 0 {
				if err := processMapping(p, func(name string) (any, bool) { return values, true }); err != nil {
					return nil, err
				}
			} else if p.Required {
				return nil, ErrRequiredFieldMissing.Wrap(fmt.Errorf("field: %s", p.Name))
			}
		} else {
			// Otherwise, maintain the old behavior of getting the first value.
			if err := processMapping(p, func(name string) (any, bool) { v := queryParams.Get(name); return v, v != "" }); err != nil {
				return nil, err
			}
		}
	}
	for _, p := range reqDef.Headers {
		if err := processMapping(p, func(name string) (any, bool) { v := headerParams.Get(name); return v, v != "" }); err != nil {
			return nil, err
		}
	}
	for _, p := range reqDef.BodyFields {
		if err := processMapping(p, func(name string) (any, bool) { v, ok, _ := utils.ExtractByPath(bodyData, name); return v, ok }); err != nil {
			return nil, err
		}
	}

	// 5. Build and set DataT objects
	dataT := msg.DataT()
	for objID, dataMap := range pendingDataT {
		defineSid, ok := objDefineSids[objID]
		if !ok {
			return nil, types.ErrInvalidConfiguration.Wrap(fmt.Errorf("MsgDefineSid not found for DataT object with id: %s", objID))
		}
		newObj, err := dataT.NewItem(defineSid, objID)
		if err != nil {
			return nil, ErrDataTItemCreationFailed.Wrap(fmt.Errorf("objId: %s, sid: %s, error: %w", objID, defineSid, err))
		}
		if err := utils.Decode(dataMap, newObj.Body()); err != nil {
			return nil, types.ErrInvalidParams.Wrap(fmt.Errorf("failed to decode data into DataT item for %s: %w", objID, err))
		}
	}

	return msg, nil
}

// setValueByDotPath is a simplified helper to set a value in a nested map.
func setValueByDotPath(data map[string]any, path string, value any) {
	parts := strings.Split(path, ".")
	current := data
	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
		} else {
			if _, ok := current[part].(map[string]any); !ok {
				current[part] = make(map[string]any)
			}
			current = current[part].(map[string]any)
		}
	}
}

// convertResponse converts the final RuleMsg to a structured HTTP response.
func (n *HttpEndpointNode) convertResponse(msg types.RuleMsg) (body map[string]any, headers map[string]string, statusCode int, err error) {
	responseBody := make(map[string]any)
	responseHeaders := make(map[string]string)
	respDef := n.nodeConfig.EndpointDefinition.Response

	// Set default status code
	statusCode = respDef.SuccessCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	if msg == nil || msg.DataT() == nil {
		// log.Warn(nil, "Cannot convert response from a nil message or a message with nil DataT.")
		return responseBody, responseHeaders, statusCode, nil
	}

	// Process body fields
	nodeCtx := registry.NewMinimalNodeCtx(n.ID())
	for _, param := range respDef.BodyFields {
		nodeCtx.Debug("Extracting response body field", "path", param.Mapping.To)
		val, found, extractErr := helper.ExtractFromMsgByPath(msg, param.Mapping.To)
		if extractErr != nil {
			return nil, nil, 0, fmt.Errorf("error extracting path %s from message: %w", param.Mapping.To, extractErr)
		}
		if found {
			nodeCtx.Debug("Successfully extracted response body field", "path", param.Mapping.To, "value_type", fmt.Sprintf("%T", val))
			setValueByDotPath(responseBody, param.Name, val)
		} else {
			nodeCtx.Warn("Response body field not found in message", "path", param.Mapping.To)
		}
	}

	// Process header fields
	for _, param := range respDef.Headers {
		val, found, extractErr := helper.ExtractFromMsgByPath(msg, param.Mapping.To)
		if extractErr != nil {
			return nil, nil, 0, fmt.Errorf("error extracting path %s from message: %w", param.Mapping.To, extractErr)
		}
		if found {
			responseHeaders[param.Name] = fmt.Sprintf("%v", val)
		}
	}

	return responseBody, responseHeaders, statusCode, nil
}
