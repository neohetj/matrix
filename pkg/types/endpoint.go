package types

// MappingInfo defines how to map an HTTP parameter to a RuleMsg.
type MappingInfo struct {
	// To is the destination path in the message, e.g., "dataT.user_obj.UserID" or "metadata.tenantId".
	To string `json:"to"`
	// DefineSID is the definition SID of the target CoreObj, required only when mapping to dataT.
	DefineSID string `json:"defineSid,omitempty"`
}

// ErrorMapping defines a map from an external protocol code/status (string)
// to a list of internal Fault codes (string).
// Key: Protocol-specific error code (e.g., "404", "500" for HTTP; "RETRY", "DLQ" for Streams).
// Value: A list of internal Fault.Code values that map to this protocol code.
type ErrorMapping map[string][]string

// HttpParam combines the definition and mapping logic for a single HTTP parameter.
type HttpParam struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Description  string `json:"description,omitempty"`
	Required     bool   `json:"required,omitempty"`
	DefaultValue any    `json:"defaultValue,omitempty"`
	// Mapping embeds the mapping rule directly within the parameter definition.
	Mapping MappingInfo `json:"mapping"`
}

// HttpEndpointDef defines the structure of an HTTP endpoint using the new V2 structures.
type HttpEndpointDef struct {
	Request  HttpRequestDef  `json:"request"`
	Response HttpResponseDef `json:"response"`
}

// HttpRequestDef defines the structure of an HTTP request.
type HttpRequestDef struct {
	DTOName     string      `json:"dtoName"`
	PathParams  []HttpParam `json:"pathParams,omitempty"`
	QueryParams []HttpParam `json:"queryParams,omitempty"`
	Headers     []HttpParam `json:"headers,omitempty"`
	// BodyFields allows defining fields for a structured request body.
	BodyFields []HttpParam `json:"bodyFields,omitempty"`
}

// HttpResponseDef defines the structure of an HTTP response.
type HttpResponseDef struct {
	DTOName         string `json:"dtoName"`
	SuccessCode     int    `json:"successCode,omitempty"`
	ErrorStatusCode int    `json:"errorStatusCode,omitempty"`
	// BodyFields defines the fields of the response body.
	BodyFields []HttpParam `json:"bodyFields,omitempty"`
	Headers    []HttpParam `json:"headers,omitempty"`
}
