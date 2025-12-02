package llm

import (
	"gitlab.com/neohet/matrix/pkg/registry"
	"gitlab.com/neohet/matrix/pkg/types"
	"gitlab.com/neohet/matrix/pkg/utils"
)

const (
	DocumentRootNodeType  = "llm_memory/documentRoot"
	DocumentChunkNodeType = "llm_memory/documentChunk"
	EntityNodeType        = "llm_memory/entity"
)

// --- Prototypes ---

var (
	documentRootNodePrototype = &DocumentRootNode{
		BaseNode: *types.NewBaseNode(DocumentRootNodeType, types.NodeDefinition{
			Name:        "Document Root",
			Description: "Represents the root of a document, acting as a container for its chunks.",
			Dimension:   "Data",
			Version:     "1.0.0",
		}),
	}
	documentChunkNodePrototype = &DocumentChunkNode{
		BaseNode: *types.NewBaseNode(DocumentChunkNodeType, types.NodeDefinition{
			Name:        "Document Chunk",
			Description: "Represents a chunk of text from a document.",
			Dimension:   "Data",
			Version:     "1.0.0",
		}),
	}
	entityNodePrototype = &EntityNode{
		BaseNode: *types.NewBaseNode(EntityNodeType, types.NodeDefinition{
			Name:        "Entity",
			Description: "Represents a named entity (e.g., person, organization, location) extracted from text.",
			Dimension:   "Data",
			Version:     "1.0.0",
		}),
	}
)

func init() {
	registry.Default.NodeManager.Register(documentRootNodePrototype)
	registry.Default.NodeManager.Register(documentChunkNodePrototype)
	registry.Default.NodeManager.Register(entityNodePrototype)
}

// --- Node Implementations ---

// DocumentRootNode represents the root of a document.
// It is a non-executable data node.
type DocumentRootNode struct {
	types.BaseNode
	types.Instance
	MatrixId string `json:"matrixId"`
	Source   string `json:"source"`
}

func (n *DocumentRootNode) New() types.Node {
	return &DocumentRootNode{BaseNode: n.BaseNode}
}

func (n *DocumentRootNode) Init(configuration types.Config) error {
	return utils.Decode(configuration, n)
}

// DocumentChunkNode represents a chunk of text from a document.
// It is a non-executable data node.
type DocumentChunkNode struct {
	types.BaseNode
	types.Instance
	MatrixId  string    `json:"matrixId"`
	Text      string    `json:"text"`
	Index     int       `json:"index"`
	Embedding []float32 `json:"embedding,omitempty"`
}

func (n *DocumentChunkNode) New() types.Node {
	return &DocumentChunkNode{BaseNode: n.BaseNode}
}

func (n *DocumentChunkNode) Init(configuration types.Config) error {
	return utils.Decode(configuration, n)
}

// EntityNode represents a named entity.
// It is a non-executable data node.
type EntityNode struct {
	types.BaseNode
	types.Instance
	MatrixId   string `json:"matrixId"`
	EntityType string `json:"entity_type"`
}

func (n *EntityNode) New() types.Node {
	return &EntityNode{BaseNode: n.BaseNode}
}

func (n *EntityNode) Init(configuration types.Config) error {
	return utils.Decode(configuration, n)
}
