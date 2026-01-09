package helper

import (
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
	// Try to use the new Asset-based extraction if possible
	val, err := message.ExtractFromMsg[any](p.Msg, name)
	if err == nil {
		return val, true, nil
	}
	// Fallback or specific error handling:
	// ExtractFromMsg returns error if not found or invalid scheme.
	// We map the error to found=false if it's just a lookup failure, but ExtractFromMsg usually errors on miss.
	// For backward compatibility behavior where "not found" is not always an error:
	return val, false, err
}

func (p RuleMsgProvider) GetAll() (any, bool, error) {
	return p.Msg, true, nil
}
