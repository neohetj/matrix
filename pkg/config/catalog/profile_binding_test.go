package catalog

import "testing"

func TestProfileBindingPolicyDeclaresGeneratedSecretResource(t *testing.T) {
	policy := &ProfileBindingPolicy{
		Supported:     true,
		ResourceKinds: []string{"secret"},
		ValueField:    "secret",
		BindingGroup:  "trusted-context-hmac",
		Generation:    &SecretGenerationPolicy{Allowed: true, Encoding: "hex", Bytes: 32},
	}
	if err := policy.Validate("secret"); err != nil {
		t.Fatal(err)
	}
	if policy.ResolvedValueField() != "secret" || !policy.MatchesSecret("secret") {
		t.Fatalf("generated secret policy not projected: %#v", policy)
	}
	if policy.Matches("secret", "") {
		t.Fatal("secret policy matched an endpoint binding")
	}
	if err := policy.Validate("string"); err == nil {
		t.Fatal("secret binding must only be available to secret values")
	}
}

func TestProfileBindingPolicyDefaultsToEndpointWithoutGeneration(t *testing.T) {
	policy := &ProfileBindingPolicy{Supported: true, ResourceKinds: []string{"database"}, Schemes: []string{"postgresql"}}
	if err := policy.Validate("url"); err != nil {
		t.Fatal(err)
	}
	if policy.ResolvedValueField() != "endpoint" || policy.MatchesSecret("database") {
		t.Fatalf("legacy profile policy changed: %#v", policy)
	}
	if !policy.Matches("database", "postgresql") {
		t.Fatal("endpoint policy did not match its declared resource")
	}
}
