package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeRejectsNilInput(t *testing.T) {
	for name, input := range map[string]any{
		"nil interface": nil,
		"nil map":       map[string]any(nil),
		"nil slice":     []any(nil),
		"nil pointer":   (*struct{ Name string })(nil),
	} {
		t.Run(name, func(t *testing.T) {
			target := struct{ Name string }{Name: "keep on error"}
			require.EqualError(t, Decode(input, &target), "input data is nil")
			require.Equal(t, "keep on error", target.Name)
		})
	}
}

func TestDecodeAcceptsEmptyCollections(t *testing.T) {
	var object struct{ Name string }
	require.NoError(t, Decode(map[string]any{}, &object))
	var items []struct{ Name string }
	require.NoError(t, Decode([]any{}, &items))
	require.Empty(t, items)
}
