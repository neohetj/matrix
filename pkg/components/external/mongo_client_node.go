package external

import (
	"context"
	"fmt"
	"sync"
	"time"

	"crypto/tls"

	"github.com/neohetj/matrix/internal/builtin/base"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/pkg/utils"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	MongoClientNodeType = "external/mongoClient"
)

var (
	MongoConnectFailed = &types.Fault{Code: cnst.CodeDBConnectFailed, Message: "failed to connect to mongodb"}
)

// MongoClientNodePrototype is the shared prototype instance for registration.
var MongoClientNodePrototype = &MongoClientNode{
	BaseNode: *types.NewBaseNode(MongoClientNodeType, types.NodeMetadata{
		Name:        "MongoDB Client",
		Description: "Provides a shared mongodb connection client (*mongo.Client).",
		Dimension:   "External",
		Tags:        []string{"external", "database", "mongo", "nosql"},
		Version:     "1.0.0",
	}),
}

func init() {
	types.DefaultRegistry.GetNodeManager().Register(MongoClientNodePrototype)
	types.DefaultRegistry.GetFaultRegistry().Register(MongoConnectFailed)
}

// MongoClientNodeConfiguration holds the configuration for the MongoClientNode.
type MongoClientNodeConfiguration struct {
	URI         string `json:"uri"`
	TLSInsecure bool   `json:"tls_insecure"`
}

// MongoClientNode is a component that provides a shared mongodb connection client (*mongo.Client).
type MongoClientNode struct {
	types.BaseNode
	types.Instance
	base.Shareable[*mongo.Client]
	nodeConfig MongoClientNodeConfiguration
	client     *mongo.Client
	closeOnce  sync.Once
}

// New creates a new instance of the MongoClientNode.
func (n *MongoClientNode) New() types.Node {
	return &MongoClientNode{
		BaseNode: n.BaseNode,
	}
}

// Init initializes the node.
func (n *MongoClientNode) Init(cfg types.ConfigMap) error {
	if err := utils.Decode(cfg, &n.nodeConfig); err != nil {
		return fmt.Errorf("failed to decode mongo client node config: %w", err)
	}

	initFunc := func() (*mongo.Client, error) {
		if n.client != nil {
			return n.client, nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		clientOpts := options.Client().ApplyURI(n.nodeConfig.URI)
		if n.nodeConfig.TLSInsecure {
			clientOpts.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})
		}

		client, err := mongo.Connect(ctx, clientOpts)
		if err != nil {
			return nil, MongoConnectFailed.Wrap(err)
		}

		// Verify connection
		if err := client.Ping(ctx, nil); err != nil {
			return nil, MongoConnectFailed.Wrap(err)
		}

		n.client = client
		return n.client, nil
	}

	// The nodePool is injected by the runtime.
	return n.Shareable.Init(nil, n.nodeConfig.URI, initFunc)
}

// OnMsg for a resource node is typically a no-op.
func (n *MongoClientNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {
	// No-op
}

// Errors returns the list of possible faults that this node can produce.
func (n *MongoClientNode) Errors() []*types.Fault {
	return append(n.Shareable.Errors(), MongoConnectFailed)
}

// Destroy closes the database connection if it was created by this node.
func (n *MongoClientNode) Destroy() {
	n.closeOnce.Do(func() {
		if n.client != nil {
			if err := n.client.Disconnect(context.Background()); err != nil {
				// Log error?
			}
		}
	})
}
