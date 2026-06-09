package validation

import (
	"encoding/json"

	"github.com/neohetj/matrix/pkg/types"
)

const DefaultReportSchemaVersion = "matrix.validation.report/v1"
const DefaultEndpointCatalogSchemaVersion = "matrix.endpoint.catalog/v1"

type Mode string

const (
	ModeReportOnly Mode = "report-only"
	ModeStrict     Mode = "strict"
)

type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

type IssueCode string

const (
	CodeDanglingConnection      IssueCode = "dangling_connection"
	CodeDuplicateNodeID         IssueCode = "duplicate_node_id"
	CodeCycleDetected           IssueCode = "cycle_detected"
	CodeUnknownNodeType         IssueCode = "unknown_node_type"
	CodeUnknownFunction         IssueCode = "unknown_function"
	CodeInvalidFunctionRelation IssueCode = "invalid_function_relation"
	CodeUnknownEndpoint         IssueCode = "unknown_endpoint"
	CodeMissingEndpointTarget   IssueCode = "missing_endpoint_target"
	CodeMissingSharedRef        IssueCode = "missing_shared_ref"
	CodeOptionalFallback        IssueCode = "optional_fallback"
	CodeInvalidEndpointIO       IssueCode = "invalid_endpoint_io"
	CodeInvalidContract         IssueCode = "invalid_contract"
	CodeLoaderFailure           IssueCode = "loader_failure"
)

type TargetKind string

const (
	TargetRuleChain  TargetKind = "rulechain"
	TargetNode       TargetKind = "node"
	TargetConnection TargetKind = "connection"
	TargetEndpoint   TargetKind = "endpoint"
	TargetFunction   TargetKind = "function"
	TargetSharedRef  TargetKind = "shared_ref"
	TargetLoader     TargetKind = "loader"
	TargetContract   TargetKind = "contract"
)

type Scope struct {
	Kind       string `json:"kind,omitempty"`
	ID         string `json:"id,omitempty"`
	SourcePath string `json:"sourcePath,omitempty"`
}

type Target struct {
	Kind       TargetKind `json:"kind,omitempty"`
	ID         string     `json:"id,omitempty"`
	Path       string     `json:"path,omitempty"`
	SourcePath string     `json:"sourcePath,omitempty"`
}

type Issue struct {
	Code     IssueCode      `json:"code"`
	Severity Severity       `json:"severity"`
	Message  string         `json:"message"`
	Target   Target         `json:"target,omitempty"`
	Details  map[string]any `json:"details,omitempty"`
}

type Report struct {
	SchemaVersion   string           `json:"schemaVersion"`
	Mode            Mode             `json:"mode"`
	Scope           Scope            `json:"scope,omitempty"`
	Issues          []Issue          `json:"issues"`
	EndpointCatalog *EndpointCatalog `json:"endpointCatalog,omitempty"`
}

type EndpointCatalog struct {
	SchemaVersion string               `json:"schemaVersion"`
	Endpoints     []EndpointDescriptor `json:"endpoints"`
}

type EndpointDescriptor struct {
	ID            string                         `json:"id"`
	Name          string                         `json:"name,omitempty"`
	Type          string                         `json:"type"`
	Protocol      string                         `json:"protocol"`
	SourcePath    string                         `json:"sourcePath,omitempty"`
	Targets       []EndpointTarget               `json:"targets,omitempty"`
	Refs          []string                       `json:"refs,omitempty"`
	HTTP          *HTTPEndpointDescriptor        `json:"http,omitempty"`
	MCP           *MCPEndpointDescriptor         `json:"mcp,omitempty"`
	Pipeline      *PipelineEndpointDescriptor    `json:"pipeline,omitempty"`
	RedisStream   *RedisStreamEndpointDescriptor `json:"redisStream,omitempty"`
	InputMapping  *types.EndpointIOPacket        `json:"inputMapping,omitempty"`
	OutputMapping *types.EndpointIOPacket        `json:"outputMapping,omitempty"`
}

type EndpointTarget struct {
	Kind     string         `json:"kind"`
	ID       string         `json:"id,omitempty"`
	Method   string         `json:"method,omitempty"`
	Path     string         `json:"path,omitempty"`
	ToolName string         `json:"toolName,omitempty"`
	StageID  string         `json:"stageId,omitempty"`
	Channel  string         `json:"channel,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type HTTPEndpointDescriptor struct {
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	Domain      string   `json:"domain,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Async       bool     `json:"async,omitempty"`
	StartNodeID string   `json:"startNodeId,omitempty"`
}

type MCPEndpointDescriptor struct {
	ServerName  string   `json:"serverName"`
	ToolNames   []string `json:"toolNames,omitempty"`
	ToolCount   int      `json:"toolCount"`
	TargetKinds []string `json:"targetKinds,omitempty"`
}

type PipelineEndpointDescriptor struct {
	ChannelManager  string                    `json:"channelManager,omitempty"`
	ExposedChannels map[string]string         `json:"exposedChannels,omitempty"`
	Stages          []PipelineStageDescriptor `json:"stages,omitempty"`
}

type PipelineStageDescriptor struct {
	ID            string `json:"id,omitempty"`
	Name          string `json:"name,omitempty"`
	ProcessorID   string `json:"processorId,omitempty"`
	ProcessorType string `json:"processorType,omitempty"`
	InputChannel  string `json:"inputChannel,omitempty"`
	OutputChannel string `json:"outputChannel,omitempty"`
}

type RedisStreamEndpointDescriptor struct {
	Stream          string `json:"stream"`
	Group           string `json:"group"`
	Consumer        string `json:"consumer,omitempty"`
	RuleChainID     string `json:"ruleChainId"`
	StartNodeID     string `json:"startNodeId,omitempty"`
	Concurrency     int    `json:"concurrency,omitempty"`
	AutoCreateGroup bool   `json:"autoCreateGroup,omitempty"`
}

func NewReport(schemaVersion string, mode Mode, scope Scope) *Report {
	if schemaVersion == "" {
		schemaVersion = DefaultReportSchemaVersion
	}
	if mode == "" {
		mode = ModeReportOnly
	}
	return &Report{
		SchemaVersion: schemaVersion,
		Mode:          mode,
		Scope:         scope,
		Issues:        []Issue{},
	}
}

func (r *Report) AddIssue(issue Issue) {
	if r == nil {
		return
	}
	if issue.Severity == "" {
		issue.Severity = SeverityError
	}
	r.Issues = append(r.Issues, issue)
}

func (r *Report) HasErrors() bool {
	return r.ErrorCount() > 0
}

func (r *Report) ErrorCount() int {
	if r == nil {
		return 0
	}
	count := 0
	for _, issue := range r.Issues {
		if issue.Severity == SeverityError {
			count++
		}
	}
	return count
}

func (r *Report) WarningCount() int {
	if r == nil {
		return 0
	}
	count := 0
	for _, issue := range r.Issues {
		if issue.Severity == SeverityWarning {
			count++
		}
	}
	return count
}

func (r *Report) ShouldBlock() bool {
	if r == nil {
		return false
	}
	return r.Mode == ModeStrict && r.HasErrors()
}

func (r Report) MarshalJSON() ([]byte, error) {
	type reportJSON struct {
		SchemaVersion   string           `json:"schemaVersion"`
		Mode            Mode             `json:"mode"`
		Scope           Scope            `json:"scope,omitempty"`
		Issues          []Issue          `json:"issues"`
		EndpointCatalog *EndpointCatalog `json:"endpointCatalog,omitempty"`
		HasErrors       bool             `json:"hasErrors"`
		ErrorCount      int              `json:"errorCount"`
		WarningCount    int              `json:"warningCount"`
	}
	return json.Marshal(reportJSON{
		SchemaVersion:   r.SchemaVersion,
		Mode:            r.Mode,
		Scope:           r.Scope,
		Issues:          r.Issues,
		EndpointCatalog: r.EndpointCatalog,
		HasErrors:       r.HasErrors(),
		ErrorCount:      r.ErrorCount(),
		WarningCount:    r.WarningCount(),
	})
}
