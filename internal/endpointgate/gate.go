// Package endpointgate decides whether a loaded active endpoint should be
// started. It is kept separate from the engine facade so the decision can be
// unit tested without constructing a full engine.
package endpointgate

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/neohetj/matrix/pkg/asset"
	"github.com/neohetj/matrix/pkg/types"
)

// Enabled reports whether the given endpoint should be started.
//
// Endpoints that do not implement types.GatedEndpoint, and endpoints whose
// expression is empty, are always started; that keeps every endpoint definition
// written before this gate existed behaving exactly as before.
//
// A non-empty expression is either a boolean literal or a "${config:///...}"
// template. Templates resolve against the supplied engine business config with
// environment-variable fallback, so an operator can switch an endpoint off
// without editing its DSL definition. Anything that fails to resolve, or that
// resolves to a non-boolean, is an error: the caller must not silently treat an
// unreadable switch as "on".
//
// The value is read once, when the engine starts its active endpoints. Changing
// it later requires a restart.
func Enabled(endpoint types.Endpoint, business types.ConfigMap) (bool, error) {
	gated, ok := endpoint.(types.GatedEndpoint)
	if !ok {
		return true, nil
	}
	return Resolve(gated.EnableExpression(), business)
}

// Resolve evaluates a single enable expression. It is exported so node authors
// and tests can check an expression without building an endpoint instance.
func Resolve(expression string, business types.ConfigMap) (bool, error) {
	trimmed := strings.TrimSpace(expression)
	if trimmed == "" {
		return true, nil
	}
	rendered, err := asset.RenderTemplate(trimmed, asset.NewAssetContext(
		asset.WithEngineConfig(business),
	))
	if err != nil {
		return false, fmt.Errorf("resolve enable expression %q: %w", expression, err)
	}
	enabled, err := strconv.ParseBool(strings.TrimSpace(rendered))
	if err != nil {
		return false, fmt.Errorf("enable expression %q resolved to non-boolean value %q", expression, rendered)
	}
	return enabled, nil
}
