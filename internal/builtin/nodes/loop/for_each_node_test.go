package loop

import (
	"testing"

	"github.com/NeohetJ/Matrix/internal/registry"
	"github.com/stretchr/testify/assert"
)

func TestForEachNode_Definition(t *testing.T) {
	// The node prototype is registered in the init() function of the main file.
	// We can retrieve it from the registry.
	node, ok := registry.Default.NodeManager.Get(ForEachNodeType)
	if !ok {
		t.Fatalf("Failed to get node type '%s' from registry", ForEachNodeType)
	}

	contract := node.DataContract()

	// Assert that the Writes field is correctly defined with URIs.
	assert.NotNil(t, contract.Writes)
	assert.Len(t, contract.Writes, 2)

	// Check for the existence of our URIs.
	expectedLoopIndexURI := "rulemsg://metadata/" + MetadataKeyLoopIndex
	expectedIsLastItemURI := "rulemsg://metadata/" + MetadataKeyIsLastItem

	assert.Contains(t, contract.Writes, expectedLoopIndexURI)
	assert.Contains(t, contract.Writes, expectedIsLastItemURI)

	// Also assert that Reads is empty.
	assert.Empty(t, contract.Reads)
}
