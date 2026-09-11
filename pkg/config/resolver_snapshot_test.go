package config

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSnapshotDoesNotReadShadowedBusiness 保持 env 已命中时不访问低优先级来源。
func TestSnapshotDoesNotReadShadowedBusiness(t *testing.T) {
	r := NewConfigResolver(WithValueSources(context.Background(), snapshotValues{"KEY": "env"}, failingConfigSource{}))
	frozen, err := r.Snapshot([]ConfigSpec{{Key: "KEY"}})
	require.NoError(t, err)
	value, _, err := Resolve[string](frozen, ConfigSpec{Key: "KEY"})
	require.NoError(t, err)
	require.Equal(t, "env", value)
}

// TestSnapshotReadsEachSourceKeyOnce 共享 alias 不得在构造一次快照时读到两个版本。
func TestSnapshotReadsEachSourceKeyOnce(t *testing.T) {
	calls := 0
	r := NewConfigResolver(WithEnvLookup(func(key string) (string, bool) {
		if key != "SHARED" {
			return "", false
		}
		calls++
		return fmt.Sprint(calls), true
	}))
	specs := []ConfigSpec{{Key: "A", Aliases: []string{"SHARED"}}, {Key: "B", Aliases: []string{"SHARED"}}}
	frozen, err := r.Snapshot(specs)
	require.NoError(t, err)
	require.Equal(t, 1, calls)
	for _, spec := range specs {
		value, _, err := Resolve[string](frozen, spec)
		require.NoError(t, err)
		require.Equal(t, "1", value)
	}
}
