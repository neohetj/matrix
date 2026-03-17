package ops

import (
	"fmt"

	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/pkg/utils"
)

func decodeNodeConfig[T any](cfg types.ConfigMap, target *T, nodeType types.NodeType) error {
	if err := utils.Decode(cfg, target); err != nil {
		return fmt.Errorf("failed to decode %s config: %w", nodeType, err)
	}
	return nil
}
