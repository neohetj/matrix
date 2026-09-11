package catalog

import (
	"context"
	"errors"
	"testing"

	matrixconfig "github.com/neohetj/matrix/pkg/config"
	"github.com/stretchr/testify/require"
)

type failingSource struct{}

func (failingSource) Lookup(context.Context, string) (any, bool, error) {
	return nil, false, errors.New("private-provider-error")
}

func TestResolveUsesNamedSourcesAndSecretEnvOnly(t *testing.T) {
	c := loadFixture(t, fixture)
	t.Setenv("TOKEN", "host-secret-must-not-be-read")
	resolved, issues := c.Resolve(context.Background(), Sources{
		Business: Values{"BACKEND": "remote", "TOKEN": "forbidden-business"},
		Explicit: Values{"TOKEN": "forbidden-explicit", "BACKEND": "local"},
	})
	require.NotEmpty(t, issues)
	require.Equal(t, "remote", resolved.Values()["BACKEND"])
	require.NotContains(t, resolved.Values(), "TOKEN")
	resolved, issues = c.Resolve(context.Background(), Sources{Env: Values{"BACKEND": "", "TOKEN": "allowed"}, Business: Values{"BACKEND": "remote"}})
	require.Empty(t, issues)
	require.Equal(t, "allowed", resolved.Values()["TOKEN"])
	_, issues = c.Resolve(context.Background(), Sources{Env: failingSource{}})
	require.Contains(t, issues.Error(), "source_read")
	require.NotContains(t, issues.Error(), "private-provider-error")
}

func TestResolvePreservesZeroFalseAliasesAndNodeScope(t *testing.T) {
	c := loadFixture(t, `version: "1"
module: sample
domain: values
items:
  - {key: COUNT, owner: sample, type: int, description: count, resolution: placeholder, required: true, default: 3, secret: false, aliases: [OLD_COUNT]}
  - {key: ENABLED, owner: sample, type: bool, description: enabled, resolution: placeholder, required: true, default: true, secret: false}
  - {key: LOCAL, owner: sample, type: string, description: local, resolution: node_explicit, secret: false}
`)
	resolved, issues := c.Resolve(context.Background(), Sources{Env: Values{"OLD_COUNT": "0", "ENABLED": false, "LOCAL": "env"}, Explicit: Values{"LOCAL": "node", "COUNT": 99}})
	require.Empty(t, issues)
	require.Equal(t, 0, resolved.Values()["COUNT"])
	require.Equal(t, false, resolved.Values()["ENABLED"])
	require.Equal(t, "node", resolved.Values()["LOCAL"])
	copy := resolved.Values()
	copy["COUNT"] = 90
	require.Equal(t, 0, resolved.Values()["COUNT"])
	// A node's explicit override does not change the catalog or another node.
	r := matrixconfig.NewConfigResolver(matrixconfig.WithEnvLookup(func(string) (string, bool) { return "env", true }))
	c2 := loadFixture(t, fixture)
	value, err := c2.ResolveString(r, "BACKEND", "node")
	require.NoError(t, err)
	require.Equal(t, "node", value)
	value, err = c2.ResolveString(r, "BACKEND", "")
	require.NoError(t, err)
	require.Equal(t, "env", value)
}

func TestAdditionalChecksCannotMutateOrEraseBaseIssues(t *testing.T) {
	c := loadFixture(t, fixture)
	resolved, issues := c.Resolve(context.Background(), Sources{Env: Values{"BACKEND": "wrong"}}, func(view View) Issues {
		v, _ := view.Lookup("BACKEND")
		require.Equal(t, "wrong", v)
		return Issues{{Code: "business_rule", Message: "business rule failed"}}
	})
	require.Equal(t, "wrong", resolved.Values()["BACKEND"])
	require.Contains(t, issues.Error(), "enum")
	require.Contains(t, issues.Error(), "business_rule")
}

func TestResolveExistingResolverUsesSameSourcePolicy(t *testing.T) {
	c := loadFixture(t, fixture)
	r := matrixconfig.NewConfigResolver(matrixconfig.WithEnvLookup(func(key string) (string, bool) {
		if key == "BACKEND" {
			return "remote", true
		}
		return "", false
	}))
	_, issues := c.ResolveWithResolver(r)
	require.Contains(t, issues.Error(), "required: TOKEN")
}
