package external

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"gitlab.com/neohet/matrix/pkg/components/base"
	"gitlab.com/neohet/matrix/pkg/registry"
	"gitlab.com/neohet/matrix/pkg/types"
	"gitlab.com/neohet/matrix/pkg/utils"
)

const (
	Neo4jClientNodeType = "external/neo4jClient"
)

// --- Prototype ---

var (
	neo4jClientNodePrototype = &Neo4jClientNode{
		BaseNode: *types.NewBaseNode(Neo4jClientNodeType, types.NodeDefinition{
			Name:        "Neo4j Client",
			Description: "A shareable client for connecting to a Neo4j database.",
			Dimension:   "External",
			Version:     "1.0.0",
		}),
	}
)

func init() {
	registry.Default.NodeManager.Register(neo4jClientNodePrototype)
}

// --- Configuration ---

// Neo4jClientNodeConfiguration holds the configuration for the Neo4j client.
type Neo4jClientNodeConfiguration struct {
	Uri      string `json:"uri"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// --- Node Implementation ---

// Neo4jClientNode is a shareable node that provides a client connection to a Neo4j database.
type Neo4jClientNode struct {
	types.BaseNode
	types.Instance
	nodeConfig Neo4jClientNodeConfiguration
	shareable  base.Shareable[neo4j.DriverWithContext]
}

// New creates a new instance of the Neo4jClientNode.
func (n *Neo4jClientNode) New() types.Node {
	return &Neo4jClientNode{BaseNode: n.BaseNode}
}

// Init initializes the node with its configuration.
func (n *Neo4jClientNode) Init(configuration types.Config) error {
	if err := utils.Decode(configuration, &n.nodeConfig); err != nil {
		return fmt.Errorf("failed to decode neo4j client node config: %w", err)
	}

	initFunc := func() (neo4j.DriverWithContext, error) {
		driver, err := neo4j.NewDriverWithContext(
			n.nodeConfig.Uri,
			neo4j.BasicAuth(n.nodeConfig.Username, n.nodeConfig.Password, ""),
		)
		if err != nil {
			return nil, err
		}
		// Verify the connection.
		err = driver.VerifyConnectivity(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to verify neo4j connectivity: %w", err)
		}
		return driver, nil
	}

	// The resource path for Shareable is the node's own ID, as it's a self-managed resource.
	return n.shareable.Init(nil, n.ID(), initFunc)
}

// GetInstance returns the shared Neo4j driver instance.
// This method makes the node a SharedNode.
func (n *Neo4jClientNode) GetInstance() (any, error) {
	return n.shareable.Get()
}

// Destroy closes the Neo4j driver connection.
func (n *Neo4jClientNode) Destroy() {
	if instance, err := n.shareable.Get(); err == nil && instance != nil {
		instance.Close(context.Background())
	}
}
