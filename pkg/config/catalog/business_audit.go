package catalog

import (
	"sort"
	"strings"

	"github.com/neohetj/matrix/pkg/types"
)

// SecretFinding 仅包含被误放入普通 business 配置的凭据键名，不持有配置值。
type SecretFinding struct {
	Key   string `json:"key"`
	Alias string `json:"alias,omitempty"`
}

// AuditBusinessSecrets 检查所有声明的凭据键及别名，不影响 env-only 运行时取值。
func AuditBusinessSecrets(definition *Catalog, business types.ConfigMap) []SecretFinding {
	if definition == nil || len(business) == 0 {
		return nil
	}
	var findings []SecretFinding
	for _, item := range definition.items {
		if !item.Secret {
			continue
		}
		seen := map[string]bool{}
		for _, key := range append([]string{item.Key}, item.Aliases...) {
			if seen[key] {
				continue
			}
			seen[key] = true
			value, found := business[key]
			if !found || value == nil {
				continue
			}
			if text, ok := value.(string); ok && strings.TrimSpace(text) == "" {
				continue
			}
			finding := SecretFinding{Key: item.Key}
			if key != item.Key {
				finding.Alias = key
			}
			findings = append(findings, finding)
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Key == findings[j].Key {
			return findings[i].Alias < findings[j].Alias
		}
		return findings[i].Key < findings[j].Key
	})
	return findings
}
