package asset_test

import (
	"github.com/neohetj/matrix/pkg/facotry"
	"github.com/neohetj/matrix/pkg/types"
)

func init() {
	types.NewNodeCtx = facotry.NewNodeCtx
	types.NewMsg = facotry.NewMsg
	types.NewDataT = facotry.NewDataT
	types.NewSubMsg = facotry.NewSubMsg
	types.NewCoreObj = facotry.NewCoreObj
	types.NewCoreObjDef = facotry.NewCoreObjDef
}
