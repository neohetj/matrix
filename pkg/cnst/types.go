package cnst

import "strings"

// MType defines the type of data, used for validation and conversion.
type MType string

// IsList checks if the data type is a list type and returns the element type.
func (d MType) IsList() (bool, string) {
	s := string(d)
	if after, found := strings.CutPrefix(s, LIST_PREFIX); found {
		return true, after
	}
	if before, found := strings.CutSuffix(s, LIST_PREFIX); found {
		return true, before
	}
	return false, ""
}

// DataFormat defines the format of the Data field in a RuleMsg.
type MFormat string

// ErrCode defines the type for standardized error codes.
type ErrCode string

// ViewType is a string enum for different rule chain visualization types.
type ViewType string
