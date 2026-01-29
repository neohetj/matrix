package types

import (
	"net/http"
)

// HttpEndpoint is a specific type of PassiveEndpoint for handling HTTP requests.
type HttpEndpoint interface {
	PassiveEndpoint
	// HandleHttpRequest handles the incoming HTTP request.
	// The implementation should write the response to the http.ResponseWriter.
	// It should return an error if any issue occurs that needs to be handled by the adapter.
	HandleHttpRequest(w http.ResponseWriter, r *http.Request, opts ...HandleOption) error
	GetHttpPath() string
	GetHttpMethod() string
	Configuration() HttpEndpointNodeConfiguration

	// GetInputMapping returns the configuration for mapping data from the HTTP request to the RuleMsg.
	// This implements part of the SubChainTrigger interface (as DataContractProvider).
	GetInputMapping() EndpointIOPacket
	// GetOutputMapping returns the configuration for mapping data from the RuleMsg to the HTTP response.
	// This implements part of the SubChainTrigger interface (as DataContractProvider).
	GetOutputMapping() EndpointIOPacket
	// GetTargetChainID returns the ID of the rule chain triggered by this endpoint.
	// This implements the SubChainTrigger interface.
	GetTargetChainID() string
}

// ServiceErrorAspect defines an interface for intercepting and transforming ServiceErrors
// before they are written to the HTTP response.
type ServiceErrorAspect interface {
	Handle(err *ServiceError) error
}

// HandleOptions holds the optional parameters for handling an HTTP request.
type HandleOptions struct {
	ExecutionID string
	Finalizer   SnapshotFinalizer
	ErrorAspect ServiceErrorAspect
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
func WithFinalizer(f SnapshotFinalizer) HandleOption {
	return func(o *HandleOptions) {
		o.Finalizer = f
	}
}

// WithErrorAspect sets the error aspect for the request.
func WithErrorAspect(aspect ServiceErrorAspect) HandleOption {
	return func(o *HandleOptions) {
		o.ErrorAspect = aspect
	}
}

// HttpEndpointNodeConfiguration holds the V2 configuration for the HttpEndpointNode.
type HttpEndpointNodeConfiguration struct {
	RuleChainID        string          `json:"ruleChainId"`
	StartNodeID        string          `json:"startNodeId,omitempty"`
	HttpMethod         string          `json:"httpMethod"`
	HttpPath           string          `json:"httpPath"`
	Description        string          `json:"description"`
	Async              bool            `json:"async,omitempty"`
	EndpointDefinition HttpEndpointDef `json:"endpointDefinition"`
	ErrorMappings      ErrorMapping    `json:"errorMappings,omitempty"`
}

// -----------------Server/Endpoint------------------

// HttpEndpointDef defines the structure of an HTTP endpoint using the new V2 structures.
type HttpEndpointDef struct {
	Request  HttpRequestDef  `json:"request"`
	Response HttpResponseDef `json:"response"`
}

// HttpRequestDef defines the structure of an HTTP request.
type HttpRequestDef struct {
	PathParams  []EndpointIOField `json:"pathParams,omitempty"`
	QueryParams EndpointIOPacket  `json:"queryParams,omitempty"`
	Headers     EndpointIOPacket  `json:"headers,omitempty"`
	// Body defines mappings for request body.
	Body EndpointIOPacket `json:"body,omitempty"`
}

// HttpResponseDef defines the structure of an HTTP response.
type HttpResponseDef struct {
	SuccessCode     int `json:"successCode,omitempty"`
	ErrorStatusCode int `json:"errorStatusCode,omitempty"`
	// Body defines mappings for response body.
	Body    EndpointIOPacket `json:"body,omitempty"`
	Headers EndpointIOPacket `json:"headers,omitempty"`
}

// -----------------Client------------------

// HttpRequestMap defines how to construct an HTTP request from a RuleMsg.
type HttpRequestMap struct {
	URL    string `json:"url"`
	Method string `json:"method"`

	// Headers defines all header mappings, both dynamic and static.
	Headers EndpointIOPacket `json:"headers,omitempty"`

	// QueryParams defines all query parameter mappings.
	QueryParams EndpointIOPacket `json:"queryParams,omitempty"`

	// Body defines the request body mappings.
	Body EndpointIOPacket `json:"body,omitempty"`

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
	Headers EndpointIOPacket `json:"headers,omitempty"`

	// Body defines how to map the response body back to the RuleMsg.
	Body EndpointIOPacket `json:"body,omitempty"`
}
