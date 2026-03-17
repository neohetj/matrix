package ops

import (
	_ "github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/types"
)

func init() {
	nodeManager := types.DefaultRegistry.GetNodeManager()
	_ = nodeManager.Register(ApplicationNodePrototype)
	_ = nodeManager.Register(ServiceNodePrototype)
	_ = nodeManager.Register(DatabaseNodePrototype)
	_ = nodeManager.Register(MessageQueueNodePrototype)
	_ = nodeManager.Register(NetworkNodePrototype)
	_ = nodeManager.Register(VolumeNodePrototype)
	_ = nodeManager.Register(RunnerNodePrototype)
	_ = nodeManager.Register(MachineNodePrototype)
}
