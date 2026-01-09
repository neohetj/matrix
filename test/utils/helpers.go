package utils

import (
	"errors"
	"fmt"

	"github.com/neohetj/matrix/internal/contract"
	"github.com/neohetj/matrix/pkg/types"
)

// NewTestRuleMsg creates a simple RuleMsg for testing.
func NewTestRuleMsg() types.RuleMsg {
	return contract.NewDefaultRuleMsg("test", "", nil, nil)
}

// GetRootError traverses the error chain and returns the root Fault.
func GetRootError(err error) *types.Fault {
	var fault *types.Fault
	if errors.As(err, &fault) {
		for {
			unwrapped := errors.Unwrap(fault)
			if unwrapped == nil {
				return fault
			}
			if nextFault, ok := unwrapped.(*types.Fault); ok {
				fault = nextFault
			} else {
				break
			}
		}
	}
	return fault
}

// MustInt is a test helper to convert any numeric type to int for comparison.
func MustInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	case int32:
		return int(n)
	case int64:
		return int(n)
	case float32:
		return int(n)
	default:
		panic(fmt.Sprintf("unexpected type %T for MustInt conversion", v))
	}
}
