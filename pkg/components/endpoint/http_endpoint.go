package endpoint

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/helper"
	"github.com/neohetj/matrix/pkg/message"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/pkg/utils"
)

const (
	HttpEndpointNodeType = "endpoint/http"
)

var (
	DefInvalidMappingFormat    = &types.Fault{Code: cnst.CodeInvalidMappingFormat, Message: "invalid mapping format"}
	DefDataTItemCreationFailed = &types.Fault{Code: cnst.CodeDataTItemCreationFailed, Message: "failed to create new DataT item"}
)

// HttpEndpoint is a specific type of PassiveEndpoint for handling HTTP requests.
type HttpEndpoint interface {
	types.PassiveEndpoint
	// HandleHttpRequest handles the incoming HTTP request.
	// The implementation should write the response to the http.ResponseWriter.
	// It should return an error if any issue occurs that needs to be handled by the adapter.
	HandleHttpRequest(w http.ResponseWriter, r *http.Request, opts ...HandleOption) error
	GetHttpPath() string
	GetHttpMethod() string
	Configuration() HttpEndpointNodeConfiguration

	// GetInputMapping returns the configuration for mapping data from the HTTP request to the RuleMsg.
	// This implements part of the SubChainTrigger interface (as DataContractProvider).
	GetInputMapping() types.EndpointIOPacket
	// GetOutputMapping returns the configuration for mapping data from the RuleMsg to the HTTP response.
	// This implements part of the SubChainTrigger interface (as DataContractProvider).
	GetOutputMapping() types.EndpointIOPacket
	// GetTargetChainID returns the ID of the rule chain triggered by this endpoint.
	// This implements the SubChainTrigger interface.
	GetTargetChainID() string
}

// HandleOptions holds the optional parameters for handling an HTTP request.
type HandleOptions struct {
	ExecutionID string
	Finalizer   types.SnapshotFinalizer
}

// HandleOption is a function that configures HandleOptions.
type HandleOption func(*HandleOptions)

// WithExecutionID sets the execution ID for the request.
func WithExecutionID(id string) HandleOption {
	return func(o *HandleOptions) {
		o.ExecutionID = id
	}
}

// WithFinalizer sets the snapshot finalizer for the request.
func WithFinalizer(f types.SnapshotFinalizer) HandleOption {
	return func(o *HandleOptions) {
		o.Finalizer = f
	}
}

// ErrorConverter defines a contract for converting Matrix's internal errors
// into application-specific error formats.
type ErrorConverter interface {
	Convert(ep HttpEndpoint, chainErr types.FailureInfo, originalErr error) error
}

// httpEndpointNodePrototype is the shared prototype instance used for registration.
var httpEndpointNodePrototype = &HttpEndpointNode{
	BaseNode: *types.NewBaseNode(HttpEndpointNodeType, types.NodeMetadata{
		Name:        "HTTP Endpoint V2",
		Description: "Receives HTTP requests and triggers a rule chain based on a unified definition.",
		Dimension:   "Endpoint",
		Tags:        []string{"endpoint", "http", "rest", "v2"},
		Version:     "2.0.0",
	}),
}

// Self-registering to the NodeManager
func init() {
	registry.Default.GetNodeManager().Register(httpEndpointNodePrototype)
	registry.Default.GetFaultRegistry().Register(
		helper.RequestDecodingFailed,
		helper.RequiredFieldMissing,
		helper.FieldConversionFailed,
		DefInvalidMappingFormat,
		DefDataTItemCreationFailed,
	)
}

// HttpEndpointNodeConfiguration holds the V2 configuration for the HttpEndpointNode.
type HttpEndpointNodeConfiguration struct {
	RuleChainID        string                `json:"ruleChainId"`
	StartNodeID        string                `json:"startNodeId,omitempty"`
	HttpMethod         string                `json:"httpMethod"`
	HttpPath           string                `json:"httpPath"`
	Description        string                `json:"description"`
	EndpointDefinition types.HttpEndpointDef `json:"endpointDefinition"`
	ErrorMappings      types.ErrorMapping    `json:"errorMappings,omitempty"`
}

// HttpEndpointNode is a component that acts as an entry point for HTTP requests.
type HttpEndpointNode struct {
	types.BaseNode
	types.Instance
	nodeConfig       HttpEndpointNodeConfiguration
	runtimePool      types.RuntimePool
	faultCodeMap     map[string]int32
	defaultErrorCode int32
}

// New creates a new instance of the node.
func (n *HttpEndpointNode) New() types.Node {
	return &HttpEndpointNode{BaseNode: n.BaseNode}
}

// Init initializes the node with its static configuration.
func (n *HttpEndpointNode) Init(config types.ConfigMap) error {
	if err := utils.Decode(config, &n.nodeConfig); err != nil {
		return types.InvalidConfiguration.Wrap(err)
	}
	if n.nodeConfig.RuleChainID == "" {
		return types.InvalidConfiguration
	}
	if n.nodeConfig.HttpMethod == "" || n.nodeConfig.HttpPath == "" {
		return types.InvalidConfiguration
	}

	n.faultCodeMap = make(map[string]int32)
	for respCodeStr, faultCodes := range n.nodeConfig.ErrorMappings {
		code, err := strconv.Atoi(respCodeStr)
		if err != nil {
			return types.InvalidConfiguration.Wrap(fmt.Errorf("invalid response code in mapping: %s", respCodeStr))
		}
		for _, fc := range faultCodes {
			n.faultCodeMap[fc] = int32(code)
		}
	}

	n.defaultErrorCode = int32(http.StatusInternalServerError)
	if n.nodeConfig.EndpointDefinition.Response.ErrorStatusCode != 0 {
		n.defaultErrorCode = int32(n.nodeConfig.EndpointDefinition.Response.ErrorStatusCode)
	}

	return nil
}

func (n *HttpEndpointNode) createServiceErrorFromMsg(msg types.RuleMsg, errStr string) *types.ServiceError {
	failureInfo := &types.FailureInfo{
		Error: errStr,
	}

	if val, ok := msg.Metadata()[types.MetaErrorNodeID]; ok {
		failureInfo.NodeID = val
	}
	if val, ok := msg.Metadata()[types.MetaErrorNodeName]; ok {
		failureInfo.NodeName = val
	}
	if val, ok := msg.Metadata()[types.MetaErrorTimestamp]; ok {
		failureInfo.Timestamp = val
	}
	if val, ok := msg.Metadata()[types.MetaErrorCode]; ok {
		failureInfo.Code = val
	}

	// Determine response code based on mapping or default
	responseCode := n.defaultErrorCode
	// Override with specific mapping if found
	if n.faultCodeMap != nil {
		if code, ok := n.faultCodeMap[failureInfo.Code]; ok {
			responseCode = code
		}
	}

	return &types.ServiceError{
		ResponseCode: responseCode,
		UserMessage:  failureInfo.Error,
		FailureInfo:  failureInfo,
	}
}

// SetRuntimePool implements the types.Endpoint interface.
func (n *HttpEndpointNode) SetRuntimePool(pool any) error {
	if p, ok := pool.(types.RuntimePool); ok {
		n.runtimePool = p
		return nil
	}
	return types.InvalidConfiguration
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

// GetInputMapping returns the configuration for mapping data from the HTTP request to the RuleMsg.
func (n *HttpEndpointNode) GetInputMapping() types.EndpointIOPacket {
	req := n.nodeConfig.EndpointDefinition.Request
	var combined types.EndpointIOPacket

	// 1. Path Params
	combined.Fields = append(combined.Fields, req.PathParams...)

	// 2. Query Params
	combined.Fields = append(combined.Fields, req.QueryParams.Fields...)

	// 3. Headers
	combined.Fields = append(combined.Fields, req.Headers.Fields...)

	// 4. Body
	combined.Fields = append(combined.Fields, req.Body.Fields...)
	combined.MapAll = req.Body.MapAll

	return combined
}

// GetOutputMapping returns the configuration for mapping data from the RuleMsg to the HTTP response.
func (n *HttpEndpointNode) GetOutputMapping() types.EndpointIOPacket {
	resp := n.nodeConfig.EndpointDefinition.Response
	var combined types.EndpointIOPacket

	// 1. Headers
	combined.Fields = append(combined.Fields, resp.Headers.Fields...)

	// 2. Body
	combined.Fields = append(combined.Fields, resp.Body.Fields...)
	combined.MapAll = resp.Body.MapAll

	return combined
}

// GetTargetChainID returns the ID of the rule chain triggered by this endpoint.
func (n *HttpEndpointNode) GetTargetChainID() string {
	return n.nodeConfig.RuleChainID
}

// ErrorResponse is the standard JSON structure for error responses.
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// writeResponse writes the HTTP response, handling both success and error cases.
// If err is provided, it writes an error response. Otherwise, it writes the success response.
func (n *HttpEndpointNode) writeResponse(w http.ResponseWriter, statusCode int, headers map[string]string, body any, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if err != nil {
		// Error case
		if statusCode == 0 {
			statusCode = http.StatusInternalServerError
		}

		response := ErrorResponse{
			Code:    statusCode,
			Message: err.Error(),
		}

		// If body contains details, we could include them
		if details, ok := body.(string); ok && details != "" {
			response.Details = details
		}

		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Success case
	for k, v := range headers {
		w.Header().Set(k, v)
	}
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(body)
}

// HandleHttpRequest is the core method that processes the incoming HTTP request.
func (n *HttpEndpointNode) HandleHttpRequest(w http.ResponseWriter, r *http.Request, opts ...HandleOption) error {
	options := &HandleOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Create the initial message
	msg := message.NewMsg(n.nodeConfig.RuleChainID, "", make(types.Metadata), nil)

	if options.ExecutionID != "" {
		msg.Metadata()[types.ExecutionIDKey] = options.ExecutionID
	}

	nodeCtx := registry.NewMinimalNodeCtx(n.ID())
	// Process all parameter types
	if err := helper.MapHttpRequestToRuleMsg(nodeCtx, msg, n.nodeConfig.EndpointDefinition.Request, r, n.nodeConfig.HttpPath); err != nil {
		return &types.ServiceError{
			ResponseCode: http.StatusBadRequest,
			UserMessage:  err.Error(),
			Cause:        err,
		}
	}

	var rt types.Runtime
	var ok bool
	if n.runtimePool != nil {
		rt, ok = n.runtimePool.Get(n.nodeConfig.RuleChainID)
	} else {
		rt, ok = registry.Default.RuntimePool.Get(n.nodeConfig.RuleChainID)
	}

	if !ok {
		return &types.ServiceError{
			ResponseCode: n.defaultErrorCode,
			UserMessage:  fmt.Sprintf("runtime not found for rule chain: %s", n.nodeConfig.RuleChainID),
		}
	}

	onEnd := func(msg types.RuleMsg, err error) {
		if options.ExecutionID != "" && options.Finalizer != nil {
			options.Finalizer.FinalizeSnapshot(options.ExecutionID)
		}
	}

	finalMsg, execErr := rt.ExecuteAndWait(r.Context(), n.nodeConfig.StartNodeID, msg, onEnd)

	if execErr != nil {
		var serviceErr *types.ServiceError
		if errors.As(execErr, &serviceErr) {
			return serviceErr
		}
		return &types.ServiceError{ResponseCode: n.defaultErrorCode, UserMessage: "internal server error", Cause: execErr}
	}

	if finalMsg != nil {
		if errStr, ok := finalMsg.Metadata()[types.MetaError]; ok {
			serviceErr := n.createServiceErrorFromMsg(finalMsg, errStr)
			n.writeResponse(w, int(serviceErr.ResponseCode), nil, nil, serviceErr)
			return serviceErr
		}
	}

	responseBody, responseHeaders, statusCode, err := helper.MapRuleMsgToHttpResponse(nodeCtx, finalMsg, n.nodeConfig.EndpointDefinition.Response)
	if err != nil {
		return &types.ServiceError{
			ResponseCode: n.defaultErrorCode,
			UserMessage:  "failed to convert response",
			Cause:        err,
		}
	}

	n.writeResponse(w, statusCode, responseHeaders, responseBody, nil)

	return nil
}
