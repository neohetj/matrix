package transform

import (
	"testing"

	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/facotry"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type objectMapperLead struct {
	ID string `json:"id"`
}

type objectMapperPatch struct {
	RejectionReasons []string `json:"rejection_reasons"`
}

func init() {
	types.NewNodeCtx = facotry.NewNodeCtx
	types.NewMsg = facotry.NewMsg
	types.CloneMsgWithDataT = facotry.CloneMsgWithDataT
	types.NewDataT = facotry.NewDataT
	types.NewSubMsg = facotry.NewSubMsg
	types.NewCoreObj = facotry.NewCoreObj
	types.NewCoreObjDef = facotry.NewCoreObjDef
}

func registerObjectMapperTestCoreObj(t *testing.T, sample any, sid string) {
	t.Helper()
	registry.Default.CoreObjRegistry.Register(types.NewCoreObjDef(sample, sid, "object mapper test coreobj"))
	t.Cleanup(func() {
		if r, ok := registry.Default.CoreObjRegistry.(*registry.DefaultCoreObjRegistry); ok {
			r.Unregister(sid)
		}
	})
}

func TestMessageValueProviderMissingFieldUsesDefaultValue(t *testing.T) {
	const (
		leadSID  = "ObjectMapperLead_V1"
		patchSID = "ObjectMapperPatch_V1"
	)
	registerObjectMapperTestCoreObj(t, &objectMapperLead{}, leadSID)
	registerObjectMapperTestCoreObj(t, &objectMapperPatch{}, patchSID)

	ctx := registry.NewMinimalNodeCtx("test-node")
	dataT := types.NewDataT()
	lead, err := dataT.NewItem(leadSID, "lead")
	require.NoError(t, err)
	require.NoError(t, lead.SetBody(&objectMapperLead{ID: "lead-1"}))

	msg := types.NewMsg("TEST", "", types.Metadata{}, dataT)
	node := &ObjectMapperNode{
		nodeConfig: ObjectMapperNodeConfiguration{
			MappingDefinition: types.EndpointIOPacket{
				Fields: []types.EndpointIOField{
					{
						Name:         "rulemsg://dataT/lead.rejection_reasons?sid=" + leadSID,
						BindPath:     "rulemsg://dataT/patch.rejection_reasons?sid=" + patchSID,
						DefaultValue: []any{},
						Type:         "array",
					},
				},
			},
		},
	}

	node.OnMsg(ctx, msg)

	patch, ok := msg.DataT().Get("patch")
	require.True(t, ok)

	body, ok := patch.Body().(*objectMapperPatch)
	require.True(t, ok)
	assert.Empty(t, body.RejectionReasons)
}

func TestMessageValueProviderOnlyFallsBackForMissingAsset(t *testing.T) {
	ctx := registry.NewMinimalNodeCtx("test-node")
	msg := types.NewMsg("TEST", "", types.Metadata{}, types.NewDataT())
	provider := &MessageValueProvider{ctx: ctx, msg: msg}

	_, found, err := provider.GetValue("rulemsg://unsupported/path")
	require.Error(t, err)
	assert.False(t, found)
}
