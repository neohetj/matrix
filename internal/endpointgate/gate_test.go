package endpointgate

import (
	"testing"

	"github.com/neohetj/matrix/pkg/types"
)

// plainEndpoint stands in for an endpoint written before the gate existed: it
// does not implement types.GatedEndpoint at all. Only the embedded interface is
// needed here; the gate never calls any endpoint method other than the optional
// EnableExpression, so leaving it nil keeps the fake honest.
type plainEndpoint struct{ types.Endpoint }

// gatedEndpoint reports a fixed enable expression.
type gatedEndpoint struct {
	types.Endpoint
	expression string
}

func (e gatedEndpoint) EnableExpression() string { return e.expression }

func TestEnabledDefaultsToStartingWhenNotGated(t *testing.T) {
	enabled, err := Enabled(plainEndpoint{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !enabled {
		t.Fatal("an endpoint that does not implement GatedEndpoint must still start")
	}
}

func TestEnabledDefaultsToStartingWhenExpressionIsEmpty(t *testing.T) {
	for _, expression := range []string{"", "   "} {
		enabled, err := Enabled(gatedEndpoint{expression: expression}, nil)
		if err != nil {
			t.Fatalf("expression %q: unexpected error: %v", expression, err)
		}
		if !enabled {
			t.Fatalf("expression %q must be treated as always-on", expression)
		}
	}
}

func TestResolveBooleanLiterals(t *testing.T) {
	cases := map[string]bool{"true": true, "false": false}
	for expression, want := range cases {
		got, err := Resolve(expression, nil)
		if err != nil {
			t.Fatalf("expression %q: unexpected error: %v", expression, err)
		}
		if got != want {
			t.Fatalf("expression %q: got %t, want %t", expression, got, want)
		}
	}
}

func TestResolveReadsEngineBusinessConfig(t *testing.T) {
	business := types.ConfigMap{
		"blokx": map[string]any{
			"fact_sync": map[string]any{"consumers_enabled": true},
		},
	}
	enabled, err := Resolve("${config:///blokx.fact_sync.consumers_enabled?scope=engine}", business)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !enabled {
		t.Fatal("expected the engine business config value to enable the endpoint")
	}
}

func TestResolveFallsBackToEnvironmentVariable(t *testing.T) {
	t.Setenv("FEATURE_CONSUMERS_ENABLED", "true")
	enabled, err := Resolve("${config:///feature.consumers_enabled?scope=engine,env}", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !enabled {
		t.Fatal("expected the environment variable fallback to enable the endpoint")
	}
}

func TestResolveUsesDefaultWhenNothingIsConfigured(t *testing.T) {
	enabled, err := Resolve("${config:///feature.absent_switch?scope=engine,env&default=false}", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if enabled {
		t.Fatal("an absent switch with default=false must not start the endpoint")
	}
}

func TestResolveFailsClosedWhenExpressionCannotBeResolved(t *testing.T) {
	// No engine config, no environment variable, and no default: the operator
	// intent is unknown, so this must be an error rather than a silent "on".
	enabled, err := Resolve("${config:///feature.absent_switch?scope=engine,env}", nil)
	if err == nil {
		t.Fatal("expected an unresolvable expression to fail")
	}
	if enabled {
		t.Fatal("a failed resolution must not report the endpoint as enabled")
	}
}

func TestResolveFailsClosedOnNonBooleanValue(t *testing.T) {
	business := types.ConfigMap{"feature": map[string]any{"switch": "sometimes"}}
	enabled, err := Resolve("${config:///feature.switch?scope=engine}", business)
	if err == nil {
		t.Fatal("expected a non-boolean value to fail")
	}
	if enabled {
		t.Fatal("a non-boolean value must not report the endpoint as enabled")
	}
}
