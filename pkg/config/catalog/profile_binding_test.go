package catalog

import "testing"

func TestProfileBindingPolicyDeclaresGeneratedCredentialSecret(t *testing.T) {
	policy := &ProfileBindingPolicy{
		Supported:     true,
		ResourceKinds: []string{"service"},
		Schemes:       []string{"hmac-sha256"},
		SourceField:   "credential_token",
		BindingKey:    "blokx-trusted-context-hmac",
		AutoGenerate:  true,
	}
	if err := policy.Validate("secret"); err != nil {
		t.Fatal(err)
	}
	if policy.BindingField() != "credential_token" || !policy.CanAutoGenerate() {
		t.Fatalf("generated secret policy not projected: %#v", policy)
	}
	if policy.SecretBindingKey("BLOKX_TRUSTED_CONTEXT_HMAC_SECRET") != "blokx-trusted-context-hmac" {
		t.Fatal("logical secret binding key not preserved")
	}
	if !policy.Matches("service", "hmac-sha256") {
		t.Fatal("generated secret policy did not match its declared resource")
	}
	if err := policy.Validate("string"); err == nil {
		t.Fatal("credential_token binding must only be available to secret values")
	}
}

func TestProfileBindingPolicyDefaultsToEndpointWithoutGeneration(t *testing.T) {
	policy := &ProfileBindingPolicy{Supported: true, ResourceKinds: []string{"database"}, Schemes: []string{"postgresql"}}
	if err := policy.Validate("url"); err != nil {
		t.Fatal(err)
	}
	if policy.BindingField() != "endpoint" || policy.CanAutoGenerate() {
		t.Fatalf("legacy profile policy changed: %#v", policy)
	}
	if policy.SecretBindingKey("SERVICE_PG_DB_URI") != "SERVICE_PG_DB_URI" {
		t.Fatal("legacy policy did not default the logical binding key to the configuration key")
	}
}
