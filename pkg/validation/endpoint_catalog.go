package validation

import (
	"sort"
	"strings"

	"github.com/neohetj/matrix/pkg/types"
)

func BuildEndpointCatalog(endpointDefs []*types.NodeDef) *EndpointCatalog {
	catalog := &EndpointCatalog{
		SchemaVersion: DefaultEndpointCatalogSchemaVersion,
		Endpoints:     []EndpointDescriptor{},
	}
	for _, endpointDef := range endpointDefs {
		if endpointDef == nil {
			continue
		}
		catalog.Endpoints = append(catalog.Endpoints, endpointDescriptorFromNodeDef(endpointDef))
	}
	sort.SliceStable(catalog.Endpoints, func(i, j int) bool {
		left := catalog.Endpoints[i]
		right := catalog.Endpoints[j]
		if left.SourcePath != right.SourcePath {
			return left.SourcePath < right.SourcePath
		}
		return left.ID < right.ID
	})
	return catalog
}

func endpointDescriptorFromNodeDef(endpointDef *types.NodeDef) EndpointDescriptor {
	descriptor := EndpointDescriptor{
		ID:         strings.TrimSpace(endpointDef.ID),
		Name:       strings.TrimSpace(endpointDef.Name),
		Type:       strings.TrimSpace(endpointDef.Type),
		Protocol:   endpointProtocol(endpointDef.Type),
		SourcePath: endpointDef.SourcePath,
		Refs:       collectConfigRefs(endpointDef.Configuration),
	}

	switch endpointDef.Type {
	case "endpoint/http":
		applyHTTPDescriptor(&descriptor, endpointDef.Configuration)
	case "endpoint/mcp":
		applyMCPDescriptor(&descriptor, endpointDef.Configuration)
	case "endpoint/pipeline":
		applyPipelineDescriptor(&descriptor, endpointDef.Configuration)
	case "endpoint/redis_stream":
		applyRedisStreamDescriptor(&descriptor, endpointDef.Configuration)
	}
	return descriptor
}

func endpointProtocol(endpointType string) string {
	switch strings.TrimSpace(endpointType) {
	case "endpoint/http":
		return "http"
	case "endpoint/mcp":
		return "mcp"
	case "endpoint/pipeline":
		return "pipeline"
	case "endpoint/redis_stream":
		return "redis_stream"
	default:
		return "unknown"
	}
}

func applyHTTPDescriptor(descriptor *EndpointDescriptor, config types.ConfigMap) {
	cfg, err := decodeConfig[types.HttpEndpointNodeConfiguration](config)
	if err != nil {
		return
	}
	descriptor.HTTP = &HTTPEndpointDescriptor{
		Method:      strings.TrimSpace(cfg.HttpMethod),
		Path:        strings.TrimSpace(cfg.HttpPath),
		Domain:      strings.TrimSpace(cfg.Domain),
		Summary:     strings.TrimSpace(cfg.Summary),
		Tags:        append([]string{}, cfg.Tags...),
		Async:       cfg.Async,
		StartNodeID: strings.TrimSpace(cfg.StartNodeID),
	}
	if targetID := strings.TrimSpace(cfg.RuleChainID); targetID != "" {
		descriptor.Targets = append(descriptor.Targets, EndpointTarget{
			Kind: "rulechain",
			ID:   targetID,
		})
	}
	input := combineHTTPRequestMapping(cfg.EndpointDefinition.Request)
	output := combineHTTPResponseMapping(cfg.EndpointDefinition.Response)
	descriptor.InputMapping = &input
	descriptor.OutputMapping = &output
}

func applyMCPDescriptor(descriptor *EndpointDescriptor, config types.ConfigMap) {
	cfg, err := decodeConfig[types.McpEndpointNodeConfiguration](config)
	if err != nil {
		return
	}
	mcp := &MCPEndpointDescriptor{
		ServerName: strings.TrimSpace(cfg.ServerName),
		ToolNames:  []string{},
		ToolCount:  len(cfg.Tools),
	}
	targetKinds := map[string]struct{}{}
	for _, tool := range cfg.Tools {
		toolName := strings.TrimSpace(tool.Name)
		if toolName != "" {
			mcp.ToolNames = append(mcp.ToolNames, toolName)
		}
		targetKind := strings.TrimSpace(tool.Target.Kind)
		if targetKind != "" {
			targetKinds[targetKind] = struct{}{}
		}
		target := EndpointTarget{
			Kind:     targetKind,
			ID:       strings.TrimSpace(tool.Target.ID),
			Method:   strings.TrimSpace(tool.Target.Method),
			Path:     strings.TrimSpace(firstNonEmpty(tool.Target.Path, tool.Target.URL)),
			ToolName: toolName,
		}
		if target.Kind != "" || target.ID != "" || target.Path != "" {
			descriptor.Targets = append(descriptor.Targets, target)
		}
	}
	mcp.TargetKinds = sortedStringKeys(targetKinds)
	descriptor.MCP = mcp
}

type pipelineConfigDescriptor struct {
	Stages          []pipelineStageConfigDescriptor `json:"stages"`
	ExposedChannels map[string]string               `json:"exposedChannels"`
	ChannelManager  string                          `json:"channelManager"`
}

type pipelineStageConfigDescriptor struct {
	Name          string                      `json:"name"`
	ID            string                      `json:"id"`
	Processor     pipelineProcessorDescriptor `json:"processor"`
	InputChannel  string                      `json:"inputChannel,omitempty"`
	OutputChannel string                      `json:"outputChannel,omitempty"`
}

type pipelineProcessorDescriptor struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

func applyPipelineDescriptor(descriptor *EndpointDescriptor, config types.ConfigMap) {
	cfg, err := decodeConfig[pipelineConfigDescriptor](config)
	if err != nil {
		return
	}
	pipeline := &PipelineEndpointDescriptor{
		ChannelManager:  strings.TrimSpace(cfg.ChannelManager),
		ExposedChannels: copyStringMap(cfg.ExposedChannels),
		Stages:          []PipelineStageDescriptor{},
	}
	seenTargets := map[string]struct{}{}
	for _, stage := range cfg.Stages {
		stageDescriptor := PipelineStageDescriptor{
			ID:            strings.TrimSpace(stage.ID),
			Name:          strings.TrimSpace(stage.Name),
			ProcessorID:   strings.TrimSpace(stage.Processor.ID),
			ProcessorType: strings.TrimSpace(stage.Processor.Type),
			InputChannel:  strings.TrimSpace(stage.InputChannel),
			OutputChannel: strings.TrimSpace(stage.OutputChannel),
		}
		pipeline.Stages = append(pipeline.Stages, stageDescriptor)
		if stageDescriptor.ProcessorID == "" {
			continue
		}
		if _, ok := seenTargets[stageDescriptor.ProcessorID]; ok {
			continue
		}
		seenTargets[stageDescriptor.ProcessorID] = struct{}{}
		descriptor.Targets = append(descriptor.Targets, EndpointTarget{
			Kind:    firstNonEmpty(stageDescriptor.ProcessorType, "rulechain"),
			ID:      stageDescriptor.ProcessorID,
			StageID: stageDescriptor.ID,
			Channel: stageDescriptor.InputChannel,
		})
	}
	descriptor.Pipeline = pipeline
}

type redisStreamConfigDescriptor struct {
	Stream          string                 `json:"stream"`
	Group           string                 `json:"group"`
	Consumer        string                 `json:"consumer,omitempty"`
	RuleChainID     string                 `json:"ruleChainId"`
	StartNodeID     string                 `json:"startNodeId,omitempty"`
	Input           types.EndpointIOPacket `json:"input,omitempty"`
	Concurrency     int                    `json:"concurrency,omitempty"`
	AutoCreateGroup bool                   `json:"autoCreateGroup,omitempty"`
}

func applyRedisStreamDescriptor(descriptor *EndpointDescriptor, config types.ConfigMap) {
	cfg, err := decodeConfig[redisStreamConfigDescriptor](config)
	if err != nil {
		return
	}
	descriptor.RedisStream = &RedisStreamEndpointDescriptor{
		Stream:          strings.TrimSpace(cfg.Stream),
		Group:           strings.TrimSpace(cfg.Group),
		Consumer:        strings.TrimSpace(cfg.Consumer),
		RuleChainID:     strings.TrimSpace(cfg.RuleChainID),
		StartNodeID:     strings.TrimSpace(cfg.StartNodeID),
		Concurrency:     cfg.Concurrency,
		AutoCreateGroup: cfg.AutoCreateGroup,
	}
	descriptor.InputMapping = &cfg.Input
	if targetID := strings.TrimSpace(cfg.RuleChainID); targetID != "" {
		descriptor.Targets = append(descriptor.Targets, EndpointTarget{
			Kind: "rulechain",
			ID:   targetID,
		})
	}
}

func combineHTTPRequestMapping(req types.HttpRequestDef) types.EndpointIOPacket {
	var packet types.EndpointIOPacket
	packet.Fields = append(packet.Fields, req.PathParams...)
	packet.Fields = append(packet.Fields, req.QueryParams.Fields...)
	packet.Fields = append(packet.Fields, req.Headers.Fields...)
	packet.Fields = append(packet.Fields, req.Body.Fields...)
	packet.MapAll = req.Body.MapAll
	return packet
}

func combineHTTPResponseMapping(resp types.HttpResponseDef) types.EndpointIOPacket {
	var packet types.EndpointIOPacket
	packet.Fields = append(packet.Fields, resp.Headers.Fields...)
	packet.Fields = append(packet.Fields, resp.Body.Fields...)
	packet.MapAll = resp.Body.MapAll
	return packet
}
