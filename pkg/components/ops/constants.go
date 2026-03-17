package ops

import (
	opsmodel "github.com/neohetj/matrix/pkg/ops/model"
	"github.com/neohetj/matrix/pkg/types"
)

const (
	ApplicationNodeType  types.NodeType = opsmodel.ApplicationNodeType
	ServiceNodeType      types.NodeType = opsmodel.ServiceNodeType
	DatabaseNodeType     types.NodeType = opsmodel.DatabaseNodeType
	MessageQueueNodeType types.NodeType = opsmodel.MessageQueueNodeType
	NetworkNodeType      types.NodeType = opsmodel.NetworkNodeType
	VolumeNodeType       types.NodeType = opsmodel.VolumeNodeType
	RunnerNodeType       types.NodeType = opsmodel.RunnerNodeType
	MachineNodeType      types.NodeType = opsmodel.MachineNodeType
)
