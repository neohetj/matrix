package types

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
