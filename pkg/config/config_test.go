package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestShouldFixedEndpointOverrideDSLOnConflict_DefaultTrue(t *testing.T) {
	cfg := NewConfig()
	if !cfg.ShouldFixedEndpointOverrideDSLOnConflict() {
		t.Fatalf("expected default fixed endpoint override behavior to be true")
	}
}

func TestShouldFixedEndpointOverrideDSLOnConflict_ExplicitValue(t *testing.T) {
	trueVal := true
	falseVal := false

	cfgTrue := MatrixConfig{
		FixedEndpoint: FixedEndpointConfig{OverrideDSLOnConflict: &trueVal},
	}
	if !cfgTrue.ShouldFixedEndpointOverrideDSLOnConflict() {
		t.Fatalf("expected explicit true to return true")
	}

	cfgFalse := MatrixConfig{
		FixedEndpoint: FixedEndpointConfig{OverrideDSLOnConflict: &falseVal},
	}
	if cfgFalse.ShouldFixedEndpointOverrideDSLOnConflict() {
		t.Fatalf("expected explicit false to return false")
	}
}

func TestShouldFixedEndpointOverrideDSLOnConflict_YAML(t *testing.T) {
	const yamlText = `
fixedEndpoint:
  overrideDslOnConflict: false
`

	var cfg MatrixConfig
	if err := yaml.Unmarshal([]byte(yamlText), &cfg); err != nil {
		t.Fatalf("yaml unmarshal failed: %v", err)
	}
	if cfg.ShouldFixedEndpointOverrideDSLOnConflict() {
		t.Fatalf("expected yaml overrideDslOnConflict=false to disable fixed override")
	}
}
