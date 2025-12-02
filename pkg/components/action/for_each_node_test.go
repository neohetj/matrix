package action

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/neohet/matrix/pkg/registry"
)

func TestForEachNode_Definition(t *testing.T) {
	// The node prototype is registered in the init() function of the main file.
	// We can retrieve it from the registry.
	node, ok := registry.Default.NodeManager.Get(ForEachNodeType)
	if !ok {
		t.Fatalf("Failed to get node type '%s' from registry", ForEachNodeType)
	}

	contract := node.GetDataContract()

	// Assert that the WritesMetadata field is correctly defined.
	assert.NotNil(t, contract.WritesMetadata)
	assert.Len(t, contract.WritesMetadata, 2)

	// Check for the existence of our keys. The order is not guaranteed.
	foundLoopIndex := false
	foundIsLastItem := false
	for _, metaDef := range contract.WritesMetadata {
		if metaDef.Key == MetadataKeyLoopIndex {
			foundLoopIndex = true
		}
		if metaDef.Key == MetadataKeyIsLastItem {
			foundIsLastItem = true
		}
	}

	assert.True(t, foundLoopIndex, "Expected to find MetadataKeyLoopIndex in WritesMetadata")
	assert.True(t, foundIsLastItem, "Expected to find MetadataKeyIsLastItem in WritesMetadata")

	// Also assert that other contracts are empty/nil.
	assert.Empty(t, contract.ReadsData)
	assert.Empty(t, contract.ReadsMetadata)
	assert.Empty(t, contract.Inputs)
	assert.Empty(t, contract.Outputs)
}
