package action

import (
	"fmt"

	"github.com/expr-lang/expr"
	"gitlab.com/neohet/matrix/pkg/helper"
	"gitlab.com/neohet/matrix/pkg/registry"
	"gitlab.com/neohet/matrix/pkg/types"
	"gitlab.com/neohet/matrix/pkg/utils"
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
	BaseNode: *types.NewBaseNode(LogNodeType, types.NodeDefinition{
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
func (n *LogNode) Init(config types.Config) error {
	if err := utils.Decode(config, &n.nodeConfig); err != nil {
		return fmt.Errorf("failed to decode log node config: %w", err)
	}
	return nil
}

// OnMsg logs the configured message.
func (n *LogNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {
	env := helper.BuildDataSource(msg)
	var logMessage string

	if len(n.nodeConfig.Args) > 0 {
		var args []any
		for _, argExpr := range n.nodeConfig.Args {
			program, err := expr.Compile(argExpr, expr.Env(env))
			if err != nil {
				ctx.Warn("Failed to compile log arg expression", "expr", argExpr, "error", err)
				args = append(args, fmt.Sprintf("<!%s!>", argExpr))
				continue
			}
			output, err := expr.Run(program, env)
			if err != nil {
				ctx.Warn("Failed to run log arg expression", "expr", argExpr, "error", err)
				args = append(args, fmt.Sprintf("<!%s!>", argExpr))
				continue
			}
			args = append(args, output)
		}
		logMessage = fmt.Sprintf(n.nodeConfig.Message, args...)
	} else {
		// Fallback to original behavior: pass env as the single argument
		logMessage = fmt.Sprintf(n.nodeConfig.Message)
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
