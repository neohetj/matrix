package mcp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestHTTPRedactionPreservesJSONAndLogLines(t *testing.T) {
	original := "access_token=private-value\nline two\nauthorization: Bearer secret-token\nline four\n"
	body, _ := json.Marshal(map[string]any{"content": original, "access_token": "another-secret", "size": 99})
	result := NewHTTPToolResult(200, body, 4096)
	var projected map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].Text), &projected); err != nil {
		t.Fatal(err)
	}
	content := projected["content"].(string)
	if strings.Count(content, "\n") != strings.Count(original, "\n") || !strings.Contains(content, "line two\n") {
		t.Fatalf("log boundaries changed: %q", content)
	}
	if strings.Contains(result.Content[0].Text, "private-value") || strings.Contains(result.Content[0].Text, "another-secret") || strings.Contains(result.Content[0].Text, "secret-token") {
		t.Fatal("secret leaked")
	}
}

func TestJSONRedactionRetainsPrefixedCredentialKeyProtection(t *testing.T) {
	body := []byte(`{"headers":{"Set-Cookie":"session=private-cookie; Path=/","X-Authorization":"private-authorization","custom_api_key":"private-key"},"count":9007199254740993}`)
	result := NewHTTPToolResult(200, body, 4096)
	for _, secret := range []string{"private-cookie", "private-authorization", "private-key"} {
		if strings.Contains(result.Content[0].Text, secret) {
			t.Fatalf("prefixed credential key leaked: %s", secret)
		}
	}
	if !strings.Contains(result.Content[0].Text, "9007199254740993") {
		t.Fatal("numeric precision changed")
	}
}
