package external

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // Register Postgres driver

	"github.com/neohetj/matrix/internal/builtin/base"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/pkg/utils"
)

const (
	SqlClientNodeType = "external/sqlClient"
)

var (
	SqlConnectFailed = &types.Fault{Code: cnst.CodeSqlConnectFailed, Message: "failed to connect to sql database"}
)

// SqlClientNodePrototype is the shared prototype instance for registration.
var SqlClientNodePrototype = &SqlClientNode{
	BaseNode: *types.NewBaseNode(SqlClientNodeType, types.NodeMetadata{
		Name:        "SQL Client",
		Description: "Provides a shared sql database connection client (*sqlx.DB).",
		Dimension:   "External",
		Tags:        []string{"external", "database", "sql", "postgres"},
		Version:     "1.0.0",
	}),
}

func init() {
	types.DefaultRegistry.GetNodeManager().Register(SqlClientNodePrototype)
	types.DefaultRegistry.GetFaultRegistry().Register(SqlConnectFailed)
}

// SqlClientNodeConfiguration holds the configuration for the SqlClientNode.
type SqlClientNodeConfiguration struct {
	DriverName string `json:"driverName"`
	URI        string `json:"uri"`
	PoolSize   int    `json:"poolSize"`
}

// SqlClientNode is a component that provides a shared sql database connection client (*sqlx.DB).
type SqlClientNode struct {
	types.BaseNode
	types.Instance
	base.Shareable[*sqlx.DB]
	nodeConfig SqlClientNodeConfiguration
	client     *sqlx.DB
	closeOnce  sync.Once
}

// New creates a new instance of the SqlClientNode.
func (n *SqlClientNode) New() types.Node {
	return &SqlClientNode{
		BaseNode: n.BaseNode,
	}
}

// Init initializes the node.
func (n *SqlClientNode) Init(cfg types.ConfigMap) error {
	if err := utils.Decode(cfg, &n.nodeConfig); err != nil {
		return fmt.Errorf("failed to decode sql client node config: %w", err)
	}

	initFunc := func() (*sqlx.DB, error) {
		if n.client != nil {
			return n.client, nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		db, err := sqlx.ConnectContext(ctx, n.nodeConfig.DriverName, n.nodeConfig.URI)
		if err != nil {
			return nil, SqlConnectFailed.Wrap(err)
		}

		if n.nodeConfig.PoolSize > 0 {
			db.SetMaxOpenConns(n.nodeConfig.PoolSize)
		}

		n.client = db
		return n.client, nil
	}

	// The nodePool is injected by the runtime.
	// Use DSN as the unique key for sharing if possible, but URI/DSN usually contains secrets.
	// Shareable.Init logic handles sharing key.
	return n.Shareable.Init(nil, n.nodeConfig.URI, initFunc)
}

// OnMsg for a resource node is typically a no-op.
func (n *SqlClientNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {
	// No-op
}

// Errors returns the list of possible faults that this node can produce.
func (n *SqlClientNode) Errors() []*types.Fault {
	return append(n.Shareable.Errors(), SqlConnectFailed)
}

// Destroy closes the database connection if it was created by this node.
func (n *SqlClientNode) Destroy() {
	n.closeOnce.Do(func() {
		if n.client != nil {
			if err := n.client.Close(); err != nil {
				// Log error?
			}
		}
	})
}
