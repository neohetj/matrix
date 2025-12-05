package builtin

import (
	"github.com/NeohetJ/Matrix/internal/builtin/base"
	_ "github.com/NeohetJ/Matrix/internal/builtin/nodes/loop"
	_ "github.com/NeohetJ/Matrix/internal/builtin/nodes/transform"
	"github.com/NeohetJ/Matrix/internal/registry"
)

func init() {
	// Register all builtin node prototypes.
	// The registry is guaranteed to be initialized at this point because
	// this package imports it directly.
	registry.Default.NodeManager.Register(base.FunctionsNodePrototype)
	registry.Default.FaultRegistry.Register(base.NodePoolNil, base.ClientNotInit)
}
