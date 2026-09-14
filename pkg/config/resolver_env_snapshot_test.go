package config

import "testing"

// TestSnapshotFreezesAbsentCanonicalEnvOnce 防止同名兼容查询覆盖首次缺失事实。
func TestSnapshotFreezesAbsentCanonicalEnvOnce(t *testing.T) {
	for _, withAlias := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing", true: "alias"}[withAlias], func(t *testing.T) {
			calls := map[string]int{}
			resolver := NewConfigResolver(WithEnvLookup(func(key string) (string, bool) {
				calls[key]++
				if key == "CLIENT_SECRET" && calls[key] > 1 {
					return "unexpected-second-read", true
				}
				if key == "LEGACY_SECRET" {
					return "alias-value", true
				}
				return "", false
			}))
			spec := ConfigSpec{Key: "CLIENT_SECRET", Secret: true}
			if withAlias {
				spec.Aliases = []string{"LEGACY_SECRET"}
			}
			frozen, err := resolver.Snapshot([]ConfigSpec{spec})
			if err != nil {
				t.Fatalf("Snapshot() error = %v", err)
			}
			if calls["CLIENT_SECRET"] != 1 {
				t.Errorf("canonical lookup count = %d, want one", calls["CLIENT_SECRET"])
			}
			for range 2 {
				value, meta, err := Resolve[string](frozen, spec)
				if err != nil {
					t.Fatalf("Resolve() error = %v", err)
				}
				if withAlias {
					if value != "alias-value" || meta.Alias != "LEGACY_SECRET" || calls["LEGACY_SECRET"] != 1 {
						t.Error("canonical absence did not select and freeze the declared alias")
					}
				} else if value != "" || meta.Source != SourceNone {
					t.Error("initial absence was replaced by a later environment value")
				}
			}
		})
	}
}

// TestSnapshotKeepsDistinctNormalizedEnvName 保留点号键的大写下划线兼容来源。
func TestSnapshotKeepsDistinctNormalizedEnvName(t *testing.T) {
	calls := map[string]int{}
	resolver := NewConfigResolver(WithEnvLookup(func(key string) (string, bool) {
		calls[key]++
		return "normalized-value", key == "DOMAIN_SECRET"
	}))
	spec := ConfigSpec{Key: "domain.secret", Secret: true}
	frozen, err := resolver.Snapshot([]ConfigSpec{spec})
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	value, _, err := Resolve[string](frozen, spec)
	if err != nil || value != "normalized-value" {
		t.Fatal("distinct normalized environment name was not preserved")
	}
	if calls["domain.secret"] != 1 || calls["DOMAIN_SECRET"] != 1 {
		t.Fatal("each distinct environment name must be read once")
	}
}
