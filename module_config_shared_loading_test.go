package matrix

import (
	"strings"
	"testing"
)

// TestModuleConfigRejectsInvalidSharedDefinitions 验证已选择加载的资源不能在初始化失败后被静默丢弃。
func TestModuleConfigRejectsInvalidSharedDefinitions(t *testing.T) {
	for _, tc := range []struct{ name, dsl, detail string }{
		{"invalid_json", `{`, "resources.json"},
		{"unknown_node", `{"metadata":{"nodes":[{"id":"broken-resource","type":"test/not-registered"}]}}`, "test/not-registered"},
		{"invalid_config", `{"metadata":{"nodes":[{"id":"broken-resource","type":"external/redisClient","configuration":{"uri":"redis://localhost/0","dialTimeout":"invalid"}}]}}`, "dialTimeout"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := lifecycleConfig(t, map[string]string{"shared/resources.json": tc.dsl})
			e, err := New(cfg, WithModuleConfig("sample", fixtureReader("value")))
			if e != nil {
				e.SharedNodePool().Stop()
			}
			if err == nil || !strings.Contains(err.Error(), tc.detail) {
				t.Fatalf("expected actionable shared resource error containing %q, got %v", tc.detail, err)
			}
		})
	}
}

// TestModuleConfigAllowsAbsentSharedDirectory 保留未声明可选资源目录的启动兼容性。
func TestModuleConfigAllowsAbsentSharedDirectory(t *testing.T) {
	cfg := lifecycleConfig(t, map[string]string{})
	e, err := New(cfg, WithModuleConfig("sample", fixtureReader("value")))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(e.SharedNodePool().Stop)
}
