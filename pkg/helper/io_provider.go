package helper

import (
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/message"
	"github.com/neohetj/matrix/pkg/types"
)

// ValueProvider defines a contract for providing values during inbound/outbound processing.
type ValueProvider interface {
	// GetValue retrieves a single value by name.
	GetValue(name string) (any, bool, error)
	// GetAll retrieves the entire data set, typically used for MapAll.
	GetAll() (any, bool, error)
}

// MapProvider implements ValueProvider for a simple map.
type MapProvider map[string]any

func (p MapProvider) GetValue(name string) (any, bool, error) {
	val, ok := p[name]
	return val, ok, nil
}

func (p MapProvider) GetAll() (any, bool, error) {
	return map[string]any(p), true, nil
}

// RuleMsgProvider implements ValueProvider for RuleMsg.
type RuleMsgProvider struct {
	Msg types.RuleMsg
}

func (p RuleMsgProvider) GetValue(name string) (any, bool, error) {
	val, err := message.ExtractFromMsg[any](p.Msg, name)
	if err != nil {
		// If the error is a Fault with the AssetNotFound code, we treat it as a non-fatal "not found" case.
		if types.IsFault(err, cnst.CodeAssetNotFound) {
			return nil, false, nil
		}
		// For all other errors, we propagate them.
		return nil, false, err
	}
	return val, true, nil
}

func (p RuleMsgProvider) GetAll() (any, bool, error) {
	return p.Msg, true, nil
}
