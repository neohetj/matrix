package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/neohetj/matrix/pkg/types"
)

const (
	DefaultProtocolVersion = "2025-11-25"
	serverVersion          = "0.1.0"
)

// ToolProvider is the MCP-facing subset implemented by endpoint/mcp adapters.
type ToolProvider interface {
	ServerName() string
	Instructions() string
	ListTools(ctx context.Context) ([]types.McpToolDefinition, error)
	CallTool(ctx context.Context, name string, arguments map[string]any) (types.McpToolResult, error)
}

// Server exposes Matrix MCP endpoints over JSON-RPC 2.0.
type Server struct {
	name         string
	instructions string
	providers    []ToolProvider
	toolOwners   map[string]ToolProvider
}

// ServerOption configures a Server.
type ServerOption func(*Server)

// WithServerName overrides the advertised MCP server name.
func WithServerName(name string) ServerOption {
	return func(s *Server) {
		if strings.TrimSpace(name) != "" {
			s.name = strings.TrimSpace(name)
		}
	}
}

// NewServer creates a JSON-RPC MCP server over one or more tool providers.
func NewServer(providers []ToolProvider, opts ...ServerOption) (*Server, error) {
	s := &Server{
		name:       "matrix-mcp",
		providers:  make([]ToolProvider, 0, len(providers)),
		toolOwners: map[string]ToolProvider{},
	}
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		s.providers = append(s.providers, provider)
		if s.name == "matrix-mcp" && strings.TrimSpace(provider.ServerName()) != "" {
			s.name = provider.ServerName()
		}
		if s.instructions == "" && strings.TrimSpace(provider.Instructions()) != "" {
			s.instructions = provider.Instructions()
		}
	}
	for _, opt := range opts {
		opt(s)
	}
	if len(s.providers) == 0 {
		return nil, errors.New("mcp server requires at least one endpoint/mcp provider")
	}
	if err := s.refreshToolOwners(context.Background()); err != nil {
		return nil, err
	}
	return s, nil
}

// ServeHTTP handles Streamable HTTP-style JSON-RPC POST requests.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONRPCHTTP(w, jsonRPCError(nil, -32700, "failed to read request body", nil))
		return
	}
	ctx := WithIncomingHTTPHeaders(r.Context(), r.Header)
	resp, ok := s.HandleMessage(ctx, body)
	if !ok {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	writeJSONRPCHTTP(w, resp)
}

// ServeStdio serves newline-delimited JSON-RPC messages over stdio.
func (s *Server) ServeStdio(ctx context.Context, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	encoder := json.NewEncoder(out)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		resp, ok := s.HandleMessage(ctx, []byte(line))
		if !ok {
			continue
		}
		var raw any
		if err := json.Unmarshal(resp, &raw); err != nil {
			return err
		}
		if err := encoder.Encode(raw); err != nil {
			return err
		}
	}
	return scanner.Err()
}

// HandleMessage processes one JSON-RPC request and returns a response payload.
// The boolean return is false for notifications.
func (s *Server) HandleMessage(ctx context.Context, payload []byte) ([]byte, bool) {
	var req jsonRPCRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return jsonRPCError(nil, -32700, "parse error", nil), true
	}
	if req.JSONRPC != "2.0" {
		return jsonRPCError(req.ID, -32600, "invalid JSON-RPC version", nil), true
	}
	if strings.TrimSpace(req.Method) == "" {
		return jsonRPCError(req.ID, -32600, "method is required", nil), true
	}

	notification := len(req.ID) == 0
	result, errObj := s.handleRequest(ctx, req)
	if notification {
		return nil, false
	}
	if errObj != nil {
		return mustMarshal(jsonRPCEnvelope{JSONRPC: "2.0", ID: req.ID, Error: errObj}), true
	}
	return mustMarshal(jsonRPCEnvelope{JSONRPC: "2.0", ID: req.ID, Result: result}), true
}

func (s *Server) handleRequest(ctx context.Context, req jsonRPCRequest) (any, *jsonRPCErrorObject) {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req.Params), nil
	case "notifications/initialized":
		return map[string]any{}, nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		result, err := s.handleToolsList(ctx)
		if err != nil {
			return nil, rpcInternalError(err)
		}
		return result, nil
	case "tools/call":
		result, err := s.handleToolsCall(ctx, req.Params)
		if err != nil {
			return nil, rpcInvalidParams(err)
		}
		return result, nil
	default:
		return nil, &jsonRPCErrorObject{Code: -32601, Message: fmt.Sprintf("method %q not found", req.Method)}
	}
}

func (s *Server) handleInitialize(params json.RawMessage) map[string]any {
	protocolVersion := DefaultProtocolVersion
	if len(params) > 0 {
		var initParams struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		if err := json.Unmarshal(params, &initParams); err == nil && strings.TrimSpace(initParams.ProtocolVersion) != "" {
			protocolVersion = strings.TrimSpace(initParams.ProtocolVersion)
		}
	}
	result := map[string]any{
		"protocolVersion": protocolVersion,
		"capabilities": map[string]any{
			"tools": map[string]any{},
		},
		"serverInfo": map[string]any{
			"name":    s.name,
			"version": serverVersion,
		},
	}
	if strings.TrimSpace(s.instructions) != "" {
		result["instructions"] = strings.TrimSpace(s.instructions)
	}
	return result
}

func (s *Server) handleToolsList(ctx context.Context) (map[string]any, error) {
	tools := make([]protocolTool, 0, len(s.toolOwners))
	for _, provider := range s.providers {
		providerTools, err := provider.ListTools(ctx)
		if err != nil {
			return nil, err
		}
		for _, tool := range providerTools {
			tools = append(tools, protocolTool{
				Name:        tool.Name,
				Title:       tool.Title,
				Description: tool.Description,
				InputSchema: inputSchemaOrEmpty(tool.InputSchema),
				Annotations: protocolAnnotations(tool),
			})
		}
	}
	return map[string]any{"tools": tools}, nil
}

func (s *Server) handleToolsCall(ctx context.Context, params json.RawMessage) (types.McpToolResult, error) {
	var callParams struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if len(params) == 0 {
		return types.McpToolResult{}, errors.New("tools/call params are required")
	}
	if err := json.Unmarshal(params, &callParams); err != nil {
		return types.McpToolResult{}, fmt.Errorf("invalid tools/call params: %w", err)
	}
	name := strings.TrimSpace(callParams.Name)
	if name == "" {
		return types.McpToolResult{}, errors.New("tools/call params.name is required")
	}
	owner, ok := s.toolOwners[name]
	if !ok {
		return errorToolResult(fmt.Sprintf("unknown tool %q", name)), nil
	}
	return owner.CallTool(ctx, name, callParams.Arguments)
}

func (s *Server) refreshToolOwners(ctx context.Context) error {
	s.toolOwners = map[string]ToolProvider{}
	for _, provider := range s.providers {
		tools, err := provider.ListTools(ctx)
		if err != nil {
			return err
		}
		for _, tool := range tools {
			name := strings.TrimSpace(tool.Name)
			if name == "" {
				return errors.New("mcp tool name is required")
			}
			if _, exists := s.toolOwners[name]; exists {
				return fmt.Errorf("duplicate mcp tool %q across providers", name)
			}
			s.toolOwners[name] = provider
		}
	}
	return nil
}

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCEnvelope struct {
	JSONRPC string              `json:"jsonrpc"`
	ID      json.RawMessage     `json:"id,omitempty"`
	Result  any                 `json:"result,omitempty"`
	Error   *jsonRPCErrorObject `json:"error,omitempty"`
}

type jsonRPCErrorObject struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type protocolTool struct {
	Name        string                  `json:"name"`
	Title       string                  `json:"title,omitempty"`
	Description string                  `json:"description,omitempty"`
	InputSchema map[string]any          `json:"inputSchema"`
	Annotations protocolToolAnnotations `json:"annotations"`
}

type protocolToolAnnotations struct {
	ReadOnlyHint    bool `json:"readOnlyHint"`
	DestructiveHint bool `json:"destructiveHint"`
	IdempotentHint  bool `json:"idempotentHint"`
	OpenWorldHint   bool `json:"openWorldHint"`
}

func protocolAnnotations(tool types.McpToolDefinition) protocolToolAnnotations {
	readOnly := normalizedRiskLevel(tool) == "read"
	return protocolToolAnnotations{
		ReadOnlyHint:    readOnly,
		DestructiveHint: !readOnly,
		IdempotentHint:  readOnly,
		OpenWorldHint:   true,
	}
}

func inputSchemaOrEmpty(schema map[string]any) map[string]any {
	if schema != nil {
		return schema
	}
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func rpcInvalidParams(err error) *jsonRPCErrorObject {
	return &jsonRPCErrorObject{Code: -32602, Message: err.Error()}
}

func rpcInternalError(err error) *jsonRPCErrorObject {
	return &jsonRPCErrorObject{Code: -32603, Message: err.Error()}
}

func jsonRPCError(id json.RawMessage, code int, message string, data any) []byte {
	return mustMarshal(jsonRPCEnvelope{
		JSONRPC: "2.0",
		ID:      id,
		Error: &jsonRPCErrorObject{
			Code:    code,
			Message: message,
			Data:    data,
		},
	})
}

func mustMarshal(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}

func writeJSONRPCHTTP(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
