package types

// TracePayloadRef describes a trace message payload that has been externalized
// out of the main trace response body.
type TracePayloadRef struct {
	ExecutionID     string `json:"executionId"`
	LogID           string `json:"logId"`
	Source          string `json:"source"`
	SizeBytes       int    `json:"sizeBytes"`
	Reason          string `json:"reason,omitempty"`
	MimeType        string `json:"mimeType,omitempty"`
	HasImagePreview bool   `json:"hasImagePreview,omitempty"`
	Externalized    bool   `json:"externalized"`
}

// TracePayloadCarrier is implemented by trace-specific RuleMsg wrappers that
// expose payload materialization metadata to API consumers.
type TracePayloadCarrier interface {
	GetTracePayload() *TracePayloadRef
}
