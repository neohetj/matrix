package validation

import "encoding/json"

const DefaultReportSchemaVersion = "matrix.validation.report/v1"

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
	CodeDanglingConnection    IssueCode = "dangling_connection"
	CodeDuplicateNodeID       IssueCode = "duplicate_node_id"
	CodeCycleDetected         IssueCode = "cycle_detected"
	CodeUnknownNodeType       IssueCode = "unknown_node_type"
	CodeUnknownFunction       IssueCode = "unknown_function"
	CodeUnknownEndpoint       IssueCode = "unknown_endpoint"
	CodeMissingEndpointTarget IssueCode = "missing_endpoint_target"
	CodeMissingSharedRef      IssueCode = "missing_shared_ref"
	CodeOptionalFallback      IssueCode = "optional_fallback"
	CodeInvalidContract       IssueCode = "invalid_contract"
	CodeLoaderFailure         IssueCode = "loader_failure"
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
	SchemaVersion string  `json:"schemaVersion"`
	Mode          Mode    `json:"mode"`
	Scope         Scope   `json:"scope,omitempty"`
	Issues        []Issue `json:"issues"`
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

func (r Report) MarshalJSON() ([]byte, error) {
	type reportJSON struct {
		SchemaVersion string  `json:"schemaVersion"`
		Mode          Mode    `json:"mode"`
		Scope         Scope   `json:"scope,omitempty"`
		Issues        []Issue `json:"issues"`
		HasErrors     bool    `json:"hasErrors"`
		ErrorCount    int     `json:"errorCount"`
		WarningCount  int     `json:"warningCount"`
	}
	return json.Marshal(reportJSON{
		SchemaVersion: r.SchemaVersion,
		Mode:          r.Mode,
		Scope:         r.Scope,
		Issues:        r.Issues,
		HasErrors:     r.HasErrors(),
		ErrorCount:    r.ErrorCount(),
		WarningCount:  r.WarningCount(),
	})
}
