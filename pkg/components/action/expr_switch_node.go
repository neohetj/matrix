package action

import (
	"fmt"
	"reflect"

	"github.com/expr-lang/expr"
	"gitlab.com/neohet/matrix/pkg/helper"
	"gitlab.com/neohet/matrix/pkg/registry"
	"gitlab.com/neohet/matrix/pkg/types"
	"gitlab.com/neohet/matrix/pkg/utils"
)

const (
	ExprSwitchNodeType = "action/exprSwitch"
)

var (
	FaultExprCompilationFailed = &types.Fault{Code: 202501001, Message: "expression compilation failed"}
	FaultExprEvaluationFailed  = &types.Fault{Code: 202501002, Message: "expression evaluation failed"}
	FaultNoMatchCase           = &types.Fault{Code: 202501003, Message: "no case matched and no default relation configured"}
)

var exprSwitchNodePrototype = &ExprSwitchNode{
	BaseNode: *types.NewBaseNode(ExprSwitchNodeType, types.NodeDefinition{
		Name:        "Expression Switch",
		Description: "Routes messages based on configurable expressions.",
		Dimension:   "Action",
		Tags:        []string{"action", "switch", "routing"},
		Version:     "1.0.0",
	}),
}

func init() {
	registry.Default.NodeManager.Register(exprSwitchNodePrototype)
	registry.Default.FaultRegistry.Register(
		FaultExprCompilationFailed,
		FaultExprEvaluationFailed,
		FaultNoMatchCase,
	)
}

// ExprSwitchNodeConfiguration holds the instance-specific configuration.
type ExprSwitchNodeConfiguration struct {
	Cases           map[string]string `json:"cases"`
	DefaultRelation string            `json:"defaultRelation,omitempty"`
}

// ExprSwitchNode routes messages based on configurable expressions.
type ExprSwitchNode struct {
	types.BaseNode
	types.Instance
	nodeConfig ExprSwitchNodeConfiguration
}

// New creates a new instance of ExprSwitchNode.
func (n *ExprSwitchNode) New() types.Node {
	return &ExprSwitchNode{
		BaseNode: n.BaseNode,
	}
}

// Type returns the node type.
func (n *ExprSwitchNode) Type() types.NodeType {
	return ExprSwitchNodeType
}

// Init initializes the node instance with its specific configuration.
func (n *ExprSwitchNode) Init(configuration types.Config) error {
	if err := utils.Decode(configuration, &n.nodeConfig); err != nil {
		return fmt.Errorf("%s: %w", types.DefInvalidConfiguration.Message, err)
	}
	if len(n.nodeConfig.Cases) == 0 {
		return fmt.Errorf("%s for node %s", types.DefInvalidConfiguration.Message, n.ID())
	}
	return nil
}

// OnMsg handles the incoming message by evaluating expressions.
func (n *ExprSwitchNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {
	ctx.Debug("ExprSwitchNode OnMsg started", "config", fmt.Sprintf("%+v", n.nodeConfig))
	env := helper.BuildDataSource(msg)

	// Custom len function for expr
	lenFunc := expr.Function("len",
		func(arguments ...any) (any, error) {
			if len(arguments) != 1 {
				return nil, fmt.Errorf("invalid number of arguments for len function")
			}
			if arguments[0] == nil {
				return 0, nil
			}
			v := reflect.ValueOf(arguments[0])
			if v.Kind() == reflect.Pointer {
				if v.IsNil() {
					return 0, nil
				}
				v = v.Elem()
			}
			switch v.Kind() {
			case reflect.Slice, reflect.Array, reflect.String, reflect.Map:
				return v.Len(), nil
			default:
				// Allow len() on non-slice types to return 0 instead of erroring
				return 0, nil
			}
		},
		new(func(any) int),
	)

	for relation, expression := range n.nodeConfig.Cases {
		if expression == "" {
			continue
		}

		program, err := expr.Compile(expression, expr.Env(env), lenFunc)
		if err != nil {
			errInfo := fmt.Errorf("expression: '%s', details: %w", expression, err)
			ctx.HandleError(msg, FaultExprCompilationFailed.Wrap(errInfo))
			return
		}

		output, err := expr.Run(program, env)
		if err != nil {
			errInfo := fmt.Errorf("expression: '%s', details: %w", expression, err)
			ctx.HandleError(msg, FaultExprEvaluationFailed.Wrap(errInfo))
			return
		}

		result, ok := output.(bool)
		ctx.Debug("Evaluated expression", "relation", relation, "expression", expression, "result", output, "isBool", ok)

		if ok && result {
			ctx.Info("Expression matched, routing to relation", "relation", relation)
			ctx.TellNext(msg, relation)
			return
		}
	}

	ctx.Debug("Checking for default relation", "defaultRelation", n.nodeConfig.DefaultRelation)
	if n.nodeConfig.DefaultRelation != "" {
		ctx.Info("No cases matched, routing to default relation", "relation", n.nodeConfig.DefaultRelation)
		ctx.TellNext(msg, n.nodeConfig.DefaultRelation)
	} else {
		ctx.Warn("No cases matched and no default relation configured")
		ctx.HandleError(msg, FaultNoMatchCase)
	}
}
