package validation

import (
	"github.com/neohetj/matrix/pkg/types"
)

type LoaderPaths struct {
	RuleChains []string
	Endpoints  []string
	Shared     []string
}

type ValidationOptions struct {
	Mode           Mode
	KnownNodeTypes []string
	Functions      []FunctionDescriptor
}

type FunctionDescriptor struct {
	ID                string
	RoutingMode       types.FunctionRoutingMode
	DeclaredRelations []string
}

func ValidateLoaderResources(provider types.ResourceProvider, paths LoaderPaths) *Report {
	return ValidateLoaderResourcesWithOptions(provider, paths, ValidationOptions{})
}

func ValidateLoaderResourcesWithOptions(provider types.ResourceProvider, paths LoaderPaths, options ValidationOptions) *Report {
	mode := options.Mode
	if mode == "" {
		mode = ModeReportOnly
	}
	report := NewReport(DefaultReportSchemaVersion, mode, Scope{
		Kind: "loader",
		ID:   providerName(provider),
	})
	if provider == nil {
		report.AddIssue(Issue{
			Code:     CodeLoaderFailure,
			Severity: SeverityError,
			Message:  "resource provider is nil",
			Target: Target{
				Kind: TargetLoader,
			},
		})
		return report
	}

	ruleChainIDs := map[string]struct{}{}
	sharedIDs := map[string]struct{}{}
	var defs []*types.RuleChainDef
	var endpointDefs []*types.NodeDef

	scanRuleChainPaths(provider, paths.RuleChains, report, ruleChainIDs, &defs)
	scanSharedPaths(provider, paths.Shared, report, sharedIDs)
	scanEndpointPaths(provider, paths.Endpoints, report, &endpointDefs)
	report.EndpointCatalog = BuildEndpointCatalog(endpointDefs)
	validateDuplicateNodeIDs(report, defs)
	validateRuleChainConnections(report, defs)
	validateRuleChainCycles(report, defs)
	validateKnownNodeTypes(report, defs, options.KnownNodeTypes)
	validateFunctionCatalog(report, defs, options.Functions)
	validateEndpointTargets(report, endpointDefs, ruleChainIDs)
	validateEndpointIOContracts(report, endpointDefs)
	validateSharedRefs(report, defs, endpointDefs, sharedIDs)

	return report
}
