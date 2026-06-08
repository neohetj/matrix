package inspection

import "github.com/neohetj/matrix/pkg/validation"

const DefaultSnapshotSchemaVersion = "matrix.inspection.snapshot/v1"

type FactKind string

const (
	FactRuleChain      FactKind = "rulechain"
	FactEndpoint       FactKind = "endpoint"
	FactFunction       FactKind = "function"
	FactSharedResource FactKind = "shared_resource"
	FactRuntime        FactKind = "runtime"
)

type RuntimeFactDescriptor struct {
	Kind       FactKind       `json:"kind"`
	ID         string         `json:"id"`
	Name       string         `json:"name,omitempty"`
	Type       string         `json:"type,omitempty"`
	SourcePath string         `json:"sourcePath,omitempty"`
	Refs       []string       `json:"refs,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type InspectionSnapshot struct {
	SchemaVersion   string                  `json:"schemaVersion"`
	EngineID        string                  `json:"engineId,omitempty"`
	RuleChains      []RuntimeFactDescriptor `json:"ruleChains"`
	Runtimes        []RuntimeFactDescriptor `json:"runtimes"`
	Endpoints       []RuntimeFactDescriptor `json:"endpoints"`
	SharedResources []RuntimeFactDescriptor `json:"sharedResources"`
	Functions       []RuntimeFactDescriptor `json:"functions"`
	Validation      *validation.Report      `json:"validation,omitempty"`
}

func NewSnapshot(schemaVersion string, engineID string) *InspectionSnapshot {
	if schemaVersion == "" {
		schemaVersion = DefaultSnapshotSchemaVersion
	}
	return &InspectionSnapshot{
		SchemaVersion:   schemaVersion,
		EngineID:        engineID,
		RuleChains:      []RuntimeFactDescriptor{},
		Runtimes:        []RuntimeFactDescriptor{},
		Endpoints:       []RuntimeFactDescriptor{},
		SharedResources: []RuntimeFactDescriptor{},
		Functions:       []RuntimeFactDescriptor{},
	}
}
