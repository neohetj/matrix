package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/neohetj/matrix/pkg/types"
)

const (
	TargetKindHTTPAPI      = "http_api"
	TargetKindExternalHTTP = "external_http"
	TargetKindRuleChain    = "rulechain"
	TargetKindHandler      = "handler"

	AuthModeDevStaticContext = "dev_static_context"
	AuthModeGatewayAssertion = "gateway_assertion"
)

var (
	placeholderPattern = regexp.MustCompile(`\$\{config:///([^}?]+)(?:\?([^}]*))?\}`)
	bearerTokenPattern = regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9._~+/=-]+`)
	secretPairPattern  = regexp.MustCompile(`(?i)(authorization|cookie|access_token|refresh_token|id_token|client_secret|internal_token|api_key)(["'\s:=]+)([^"'\s,}&]+)`)
	forbiddenArgKeys   = map[string]struct{}{
		"authorization":              {},
		"cookie":                     {},
		"company_id":                 {},
		"current_team_ids":           {},
		"identityx_current_team_ids": {},
		"identityx_permissions":      {},
		"identityx_roles":            {},
		"internal_token":             {},
		"permissions":                {},
		"roles":                      {},
		"session_id":                 {},
		"team_ids":                   {},
		"user_id":                    {},
	}
)

// Option configures an Endpoint.
type Option func(*Endpoint)

// WithHTTPClient overrides the default HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(e *Endpoint) {
		if client != nil {
			e.httpClient = client
		}
	}
}

// WithConfigValues supplies host configuration values used by config:/// placeholders.
func WithConfigValues(values map[string]any) Option {
	return func(e *Endpoint) {
		for k, v := range values {
			e.configValues[k] = v
		}
	}
}

// Endpoint adapts a Matrix endpoint/mcp configuration into MCP tool semantics.
type Endpoint struct {
	cfg          types.McpEndpointNodeConfiguration
	httpClient   *http.Client
	configValues map[string]any
	toolsByName  map[string]types.McpToolDefinition
	dispatcher   TargetDispatcher
}

// NewEndpoint creates a validated MCP endpoint adapter.
func NewEndpoint(cfg types.McpEndpointNodeConfiguration, opts ...Option) (*Endpoint, error) {
	e := &Endpoint{
		cfg:          cfg,
		configValues: map[string]any{},
		toolsByName:  map[string]types.McpToolDefinition{},
	}
	timeout := time.Duration(cfg.HTTP.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	e.httpClient = &http.Client{Timeout: timeout}
	for _, opt := range opts {
		opt(e)
	}
	if err := e.validate(); err != nil {
		return nil, err
	}
	return e, nil
}

// ServerName returns the MCP server name advertised for this endpoint.
func (e *Endpoint) ServerName() string {
	if e == nil {
		return ""
	}
	return strings.TrimSpace(e.cfg.ServerName)
}

// Instructions returns optional MCP server instructions.
func (e *Endpoint) Instructions() string {
	if e == nil {
		return ""
	}
	return strings.TrimSpace(e.cfg.Instructions)
}

// Configuration returns a copy of the endpoint configuration.
func (e *Endpoint) Configuration() types.McpEndpointNodeConfiguration {
	if e == nil {
		return types.McpEndpointNodeConfiguration{}
	}
	return e.cfg
}

// ListTools returns the module-owned MCP tool catalog.
func (e *Endpoint) ListTools(ctx context.Context) ([]types.McpToolDefinition, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if e == nil {
		return nil, errors.New("mcp endpoint is nil")
	}
	tools := make([]types.McpToolDefinition, 0, len(e.cfg.Tools))
	for _, tool := range e.cfg.Tools {
		tools = append(tools, tool)
	}
	return tools, nil
}

// CallTool invokes a configured tool through its declared target.
func (e *Endpoint) CallTool(ctx context.Context, name string, arguments map[string]any) (types.McpToolResult, error) {
	if e == nil {
		return types.McpToolResult{}, errors.New("mcp endpoint is nil")
	}
	tool, ok := e.toolsByName[strings.TrimSpace(name)]
	if !ok {
		return errorToolResult(fmt.Sprintf("unknown tool %q", name)), nil
	}
	if arguments == nil {
		arguments = map[string]any{}
	}
	if err := rejectForbiddenArguments(arguments); err != nil {
		return errorToolResult(err.Error()), nil
	}
	authContext, err := e.resolveAuthContext(ctx, tool.AuthContext)
	if err != nil {
		return errorToolResult(err.Error()), nil
	}

	switch strings.TrimSpace(tool.Target.Kind) {
	case TargetKindHTTPAPI, TargetKindExternalHTTP:
		return e.callHTTP(ctx, tool, arguments, authContext)
	case TargetKindRuleChain, TargetKindHandler:
		return e.dispatchTarget(ctx, tool, arguments, authContext)
	default:
		return errorToolResult(fmt.Sprintf("unsupported target kind %q", tool.Target.Kind)), nil
	}
}

func (e *Endpoint) validate() error {
	if strings.TrimSpace(e.cfg.ServerName) == "" {
		return errors.New("mcp endpoint serverName is required")
	}
	if strings.TrimSpace(e.cfg.ToolCatalog) != "" && len(e.cfg.Tools) > 0 {
		return errors.New("mcp endpoint cannot define both toolCatalog and inline tools")
	}
	if strings.TrimSpace(e.cfg.ToolCatalog) != "" {
		return errors.New("mcp endpoint toolCatalog is reserved for a future shared-node catalog")
	}
	seen := map[string]struct{}{}
	for _, tool := range e.cfg.Tools {
		name := strings.TrimSpace(tool.Name)
		if name == "" {
			return errors.New("mcp tool name is required")
		}
		if _, ok := seen[name]; ok {
			return fmt.Errorf("duplicate mcp tool %q", name)
		}
		seen[name] = struct{}{}
		if !isReadOnlyTool(tool) {
			return fmt.Errorf("mcp tool %q must declare riskLevel=read for MVP", name)
		}
		if err := validateInputSchema(name, tool.InputSchema); err != nil {
			return err
		}
		if strings.TrimSpace(tool.Target.Kind) == "" {
			return fmt.Errorf("mcp tool %q target.kind is required", name)
		}
		e.toolsByName[name] = tool
	}
	return nil
}

func (e *Endpoint) callHTTP(ctx context.Context, tool types.McpToolDefinition, args map[string]any, authContext ResolvedAuthContext) (types.McpToolResult, error) {
	method, targetURL, err := e.resolveHTTPTarget(tool)
	if err != nil {
		return errorToolResult(err.Error()), nil
	}

	var body io.Reader
	if method != http.MethodGet && method != http.MethodHead {
		payload, err := json.Marshal(args)
		if err != nil {
			return types.McpToolResult{}, err
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, targetURL, body)
	if err != nil {
		return types.McpToolResult{}, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	applyResolvedAuthContextHeaders(req, authContext)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return errorToolResult(err.Error()), nil
	}
	defer resp.Body.Close()

	limit := tool.Output.MaxBytes
	readLimit := limit
	if readLimit <= 0 {
		readLimit = 1 << 20
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, readLimit+1))
	if err != nil {
		return types.McpToolResult{}, err
	}
	return NewHTTPToolResult(resp.StatusCode, data, limit), nil
}

func (e *Endpoint) resolveHTTPTarget(tool types.McpToolDefinition) (string, string, error) {
	method := strings.ToUpper(strings.TrimSpace(tool.Target.Method))
	path := strings.TrimSpace(tool.Target.Path)
	if method == "" || path == "" {
		idMethod, idPath := splitTargetID(tool.Target.ID)
		if method == "" {
			method = idMethod
		}
		if path == "" {
			path = idPath
		}
	}
	if method == "" {
		method = http.MethodGet
	}

	rawURL := strings.TrimSpace(tool.Target.URL)
	if rawURL != "" {
		resolved, err := e.resolveString(rawURL)
		if err != nil {
			return "", "", err
		}
		if _, err := url.ParseRequestURI(resolved); err != nil {
			return "", "", fmt.Errorf("invalid MCP tool target url: %w", err)
		}
		return method, resolved, nil
	}
	if path == "" {
		return "", "", fmt.Errorf("mcp tool %q must define target.path, target.url, or target.id", tool.Name)
	}

	baseURL := strings.TrimSpace(e.cfg.HTTP.BaseURL)
	if baseURL == "" {
		baseURL = strings.TrimSpace(tool.Target.Params["baseURL"])
	}
	if baseURL == "" {
		return "", "", fmt.Errorf("mcp tool %q requires http.baseURL or target.url", tool.Name)
	}
	resolvedBase, err := e.resolveString(baseURL)
	if err != nil {
		return "", "", err
	}
	resolvedPath, err := e.resolveString(path)
	if err != nil {
		return "", "", err
	}
	u, err := url.Parse(strings.TrimRight(resolvedBase, "/"))
	if err != nil {
		return "", "", fmt.Errorf("invalid MCP http.baseURL: %w", err)
	}
	ref, err := url.Parse("/" + strings.TrimLeft(resolvedPath, "/"))
	if err != nil {
		return "", "", fmt.Errorf("invalid MCP tool target path: %w", err)
	}
	return method, u.ResolveReference(ref).String(), nil
}

func (e *Endpoint) resolveAuthContext(ctx context.Context, authContextName string) (ResolvedAuthContext, error) {
	name := strings.TrimSpace(authContextName)
	if name == "" {
		return ResolvedAuthContext{}, nil
	}
	ctxDef, ok := e.cfg.AuthContexts[name]
	if !ok {
		return ResolvedAuthContext{}, fmt.Errorf("mcp authContext %q is not defined", name)
	}
	resolved := ResolvedAuthContext{
		Name:    name,
		Mode:    strings.TrimSpace(ctxDef.Mode),
		Headers: map[string]string{},
	}
	switch resolved.Mode {
	case AuthModeDevStaticContext:
		for key, value := range ctxDef.Headers {
			headerName := strings.TrimSpace(key)
			if headerName == "" {
				continue
			}
			resolvedValue, err := e.resolveString(value)
			if err != nil {
				return ResolvedAuthContext{}, err
			}
			if resolvedValue == "" {
				continue
			}
			resolved.Headers[headerName] = resolvedValue
		}
	case AuthModeGatewayAssertion:
		incomingHeaders, ok := IncomingHTTPHeadersFromContext(ctx)
		if !ok {
			return ResolvedAuthContext{}, fmt.Errorf("mcp authContext %q mode %q requires incoming HTTP request headers", name, resolved.Mode)
		}
		for key, value := range ctxDef.Headers {
			headerName := strings.TrimSpace(key)
			if headerName == "" {
				continue
			}
			sourceName := headerName
			if strings.TrimSpace(value) != "" {
				resolvedSourceName, err := e.resolveString(value)
				if err != nil {
					return ResolvedAuthContext{}, err
				}
				sourceName = strings.TrimSpace(resolvedSourceName)
			}
			if sourceName == "" {
				sourceName = headerName
			}
			sourceValue := strings.TrimSpace(incomingHeaders.Get(sourceName))
			if sourceValue == "" {
				continue
			}
			resolved.Headers[headerName] = sourceValue
		}
	default:
		return ResolvedAuthContext{}, fmt.Errorf("mcp authContext %q mode %q is not supported", name, ctxDef.Mode)
	}
	return resolved, nil
}

func applyResolvedAuthContextHeaders(req *http.Request, authContext ResolvedAuthContext) {
	if req == nil || len(authContext.Headers) == 0 {
		return
	}
	for key, value := range authContext.Headers {
		headerName := strings.TrimSpace(key)
		if headerName == "" || strings.TrimSpace(value) == "" {
			continue
		}
		req.Header.Set(headerName, value)
	}
}

func (e *Endpoint) dispatchTarget(ctx context.Context, tool types.McpToolDefinition, args map[string]any, authContext ResolvedAuthContext) (types.McpToolResult, error) {
	if e.dispatcher == nil {
		return errorToolResult(fmt.Sprintf("%s targets require a runtime-bound module dispatcher and are not enabled for this MCP endpoint", strings.TrimSpace(tool.Target.Kind))), nil
	}
	result, handled, err := e.dispatcher.Dispatch(ctx, DispatchRequest{
		Tool:           tool,
		Arguments:      args,
		AuthContext:    authContext,
		EndpointConfig: e.cfg,
	})
	if !handled {
		return errorToolResult(fmt.Sprintf("unsupported target kind %q", tool.Target.Kind)), nil
	}
	if err != nil {
		return errorToolResult(err.Error()), nil
	}
	return result, nil
}

func (e *Endpoint) resolveString(raw string) (string, error) {
	result := raw
	var firstErr error
	result = placeholderPattern.ReplaceAllStringFunc(result, func(match string) string {
		if firstErr != nil {
			return match
		}
		parts := placeholderPattern.FindStringSubmatch(match)
		if len(parts) != 3 {
			firstErr = fmt.Errorf("invalid config placeholder %q", match)
			return match
		}
		key, err := url.QueryUnescape(parts[1])
		if err != nil {
			firstErr = err
			return match
		}
		query, err := url.ParseQuery(parts[2])
		if err != nil {
			firstErr = err
			return match
		}
		if value, ok := e.lookupConfigValue(key); ok {
			return value
		}
		if defaults, ok := query["default"]; ok && len(defaults) > 0 {
			return defaults[0]
		}
		firstErr = fmt.Errorf("config value %q is required", key)
		return match
	})
	if firstErr != nil {
		return "", firstErr
	}
	return result, nil
}

func (e *Endpoint) lookupConfigValue(key string) (string, bool) {
	if e != nil && e.configValues != nil {
		if value, ok := e.configValues[key]; ok && value != nil {
			return fmt.Sprint(value), true
		}
	}
	return os.LookupEnv(key)
}

func splitTargetID(id string) (string, string) {
	fields := strings.Fields(strings.TrimSpace(id))
	if len(fields) < 2 {
		return "", ""
	}
	method := strings.ToUpper(strings.TrimSpace(fields[0]))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead:
		return method, strings.TrimSpace(strings.Join(fields[1:], " "))
	default:
		return "", ""
	}
}

func isReadOnlyTool(tool types.McpToolDefinition) bool {
	return strings.EqualFold(strings.TrimSpace(tool.RiskLevel), "read")
}

func validateInputSchema(toolName string, schema map[string]any) error {
	forbidden := collectForbiddenKeys(schema)
	if len(forbidden) > 0 {
		return fmt.Errorf("mcp tool %q inputSchema contains forbidden security context fields: %s", toolName, strings.Join(forbidden, ", "))
	}
	return nil
}

func collectForbiddenKeys(value any) []string {
	seen := map[string]struct{}{}
	var walk func(any)
	walk = func(v any) {
		switch typed := v.(type) {
		case map[string]any:
			for key, child := range typed {
				normalized := normalizeSecurityKey(key)
				if _, ok := forbiddenArgKeys[normalized]; ok || strings.HasPrefix(normalized, "x_identityx_") {
					seen[key] = struct{}{}
				}
				walk(child)
			}
		case []any:
			for _, child := range typed {
				walk(child)
			}
		}
	}
	walk(value)
	out := make([]string, 0, len(seen))
	for key := range seen {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func rejectForbiddenArguments(arguments map[string]any) error {
	forbidden := collectForbiddenKeys(arguments)
	if len(forbidden) > 0 {
		return fmt.Errorf("mcp tool arguments must not provide security context field %q", forbidden[0])
	}
	return nil
}

func normalizeSecurityKey(key string) string {
	key = strings.TrimSpace(strings.ToLower(key))
	key = strings.ReplaceAll(key, "-", "_")
	key = strings.ReplaceAll(key, ".", "_")
	key = strings.ReplaceAll(key, " ", "_")
	return key
}

func textToolResult(text string) types.McpToolResult {
	return types.McpToolResult{
		Content: []types.McpToolContent{{Type: "text", Text: text}},
	}
}

func errorToolResult(text string) types.McpToolResult {
	result := textToolResult(sanitizeToolText(text))
	result.IsError = true
	return result
}

func sanitizeToolText(text string) string {
	if text == "" {
		return text
	}
	text = bearerTokenPattern.ReplaceAllString(text, "Bearer [redacted]")
	return secretPairPattern.ReplaceAllString(text, "${1}${2}[redacted]")
}
