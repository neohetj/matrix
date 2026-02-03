package pipeline

import (
	"fmt"
	"sync"

	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/types"
)

const (
	ChannelManagerNodeType = "resource/channel_manager"
)

var channelManagerNodePrototype = &ChannelManagerNode{
	BaseNode: *types.NewBaseNode(ChannelManagerNodeType, types.NodeMetadata{
		Name:        "Channel Manager",
		Description: "Manages channels for pipeline communication.",
		Dimension:   "Resource",
		Tags:        []string{"resource", "channel", "manager"},
		Version:     "1.0.0",
	}),
}

func init() {
	registry.Default.GetNodeManager().Register(channelManagerNodePrototype)
}

type ChannelManagerNode struct {
	types.BaseNode
	types.Instance
	manager *ChannelManager
}

func (n *ChannelManagerNode) New() types.Node {
	return &ChannelManagerNode{BaseNode: n.BaseNode}
}

func (n *ChannelManagerNode) Init(config types.ConfigMap) error {
	// Initialize the manager
	n.manager = &ChannelManager{
		channels: make(map[string]chan types.RuleMsg),
	}
	return nil
}

func (n *ChannelManagerNode) GetInstance() (any, error) {
	if n.manager == nil {
		return nil, types.InvalidConfiguration
	}
	return n.manager, nil
}

// ChannelManager handles the registry of channels for pipelines.
type ChannelManager struct {
	channels map[string]chan types.RuleMsg
	mu       sync.RWMutex
}

func (m *ChannelManager) Register(pipelineID, channelName string, ch chan types.RuleMsg) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s:%s", pipelineID, channelName)
	m.channels[key] = ch
}

func (m *ChannelManager) Unregister(pipelineID, channelName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s:%s", pipelineID, channelName)
	delete(m.channels, key)
}

func (m *ChannelManager) Get(pipelineID, channelName string) (chan types.RuleMsg, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := fmt.Sprintf("%s:%s", pipelineID, channelName)
	if ch, ok := m.channels[key]; ok {
		return ch, nil
	}
	return nil, fmt.Errorf("channel %s not found", key)
}
