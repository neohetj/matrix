package helper_test

import (
	"testing"

	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/facotry"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/stretchr/testify/require"
)

func init() {
	types.NewNodeCtx = facotry.NewNodeCtx
	types.NewMsg = facotry.NewMsg
	types.CloneMsgWithDataT = facotry.CloneMsgWithDataT
	types.NewDataT = facotry.NewDataT
	types.NewSubMsg = facotry.NewSubMsg
	types.NewCoreObj = facotry.NewCoreObj
	types.NewCoreObjDef = facotry.NewCoreObjDef
}

// setupTestMsg creates a message with pre-populated dataT objects for testing.
func setupTestMsg(t *testing.T) types.RuleMsg {
	dataT := types.NewDataT()

	headersObj, err := dataT.NewItem(cnst.SID_MAP_STRING_STRING, "headersObj")
	require.NoError(t, err)
	*(headersObj.Body().(*map[string]string)) = map[string]string{"X-Dynamic-Header": "dynamic-value"}

	bodyObj, err := dataT.NewItem(cnst.SID_MAP_STRING_INTERFACE, "bodyObj")
	require.NoError(t, err)
	*(bodyObj.Body().(*map[string]interface{})) = map[string]interface{}{"user": "test", "id": 123}

	queryObj, err := dataT.NewItem(cnst.SID_MAP_STRING_STRING, "queryObj")
	require.NoError(t, err)
	*(queryObj.Body().(*map[string]string)) = map[string]string{"q": "matrix", "limit": "10"}

	msg := types.NewMsg("TEST", `{"key":"value"}`, make(map[string]string), dataT)
	msg.Metadata()["requestId"] = "req-123"
	return msg
}
