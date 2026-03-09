package asset_test

import (
	"testing"

	"github.com/neohetj/matrix/pkg/asset"
	"github.com/stretchr/testify/assert"
)

func TestValidateURI_RuleMsgDataTCandidates_AcceptsValidValue(t *testing.T) {
	err := asset.ValidateURI("rulemsg://dataT/route_pipeline_id?sid=String&candidates=ep-a,ep-b")
	assert.NoError(t, err)
}

func TestValidateURI_RuleMsgDataTCandidates_RejectsUnsupportedQueryKey(t *testing.T) {
	err := asset.ValidateURI("rulemsg://dataT/route_pipeline_id?sid=String&possibleValues=ep-a")
	assert.Error(t, err)
	assert.ErrorContains(t, err, "unsupported query key")
}

func TestValidateURI_RuleMsgDataTCandidates_RejectsEmptyCandidates(t *testing.T) {
	err := asset.ValidateURI("rulemsg://dataT/route_pipeline_id?sid=String&candidates=,,")
	assert.Error(t, err)
	assert.ErrorContains(t, err, "invalid candidates")
}

func TestParseRuleMsgCandidates_DeduplicatesAndSplits(t *testing.T) {
	candidates, err := asset.ParseRuleMsgCandidates("ep-a,ep-b;ep-a|ep-c")
	assert.NoError(t, err)
	assert.Equal(t, []string{"ep-a", "ep-b", "ep-c"}, candidates)
}
