package action

import (
	"fmt"

	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/message"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/pkg/utils"
)

const (
	LogNodeType = "action/log"

	LogLevelDebug = "debug"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
	LogLevelInfo  = "info"
)

// logNodePrototype is the shared prototype instance for registration.
var logNodePrototype = &LogNode{
	BaseNode: *types.NewBaseNode(LogNodeType, types.NodeMetadata{
		Name:        "Log",
		Description: "Logs a message to the console with a specified log level.",
		Dimension:   "Action",
		Tags:        []string{"action", "log"},
		Version:     "1.0.0",
	}),
}

func init() {
	registry.Default.NodeManager.Register(logNodePrototype)
}

// LogNodeConfiguration holds the configuration for the LogNode.
type LogNodeConfiguration struct {
	Level   string   `json:"level"`
	Message string   `json:"message"`
	Args    []string `json:"args"`
}

// LogNode is a simple node that logs a message.
type LogNode struct {
	types.BaseNode
	types.Instance
	nodeConfig LogNodeConfiguration
}

// New creates a new instance of LogNode.
func (n *LogNode) New() types.Node {
	return &LogNode{BaseNode: n.BaseNode}
}

// Init initializes the node with the given configuration.
func (n *LogNode) Init(config types.ConfigMap) error {
	if err := utils.Decode(config, &n.nodeConfig); err != nil {
		return fmt.Errorf("failed to decode log node config: %w", err)
	}
	return nil
}

// OnMsg logs the configured message.
func (n *LogNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {
	// Use asset.RenderTemplate to replace placeholders in the message.
	// This replaces the deprecated BuildDataSource and expr-based argument processing.
	logMessage, err := message.ReplaceRuleMsg(n.nodeConfig.Message, msg)
	if err != nil {
		ctx.Warn("Failed to render log message template", "error", err)
		logMessage = n.nodeConfig.Message // Fallback to raw message
	}

	switch n.nodeConfig.Level {
	case LogLevelDebug:
		ctx.Debug(logMessage)
	case LogLevelWarn:
		ctx.Warn(logMessage)
	case LogLevelError:
		ctx.Error(logMessage)
	default:
		ctx.Info(logMessage)
	}
	ctx.TellSuccess(msg)
}
