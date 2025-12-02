package endpoint

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"gitlab.com/neohet/matrix/internal/log"
	"gitlab.com/neohet/matrix/pkg/registry"
	"gitlab.com/neohet/matrix/pkg/types"
	"gitlab.com/neohet/matrix/pkg/utils"
)

const (
	InotifyNodeType = "endpoint/inotify"
)

// inotifyNodePrototype is the shared prototype instance for registration.
var inotifyNodePrototype = &InotifyNode{
	BaseNode: *types.NewBaseNode(InotifyNodeType, types.NodeDefinition{
		Name:        "Inotify Endpoint",
		Description: "Actively listens for file system events (e.g., write, create).",
		Dimension:   "Endpoint",
		Tags:        []string{"endpoint", "file"},
		Version:     "1.0.0",
	}),
}

// Self-registering to the NodeManager
func init() {
	registry.Default.NodeManager.Register(inotifyNodePrototype)
}

// InotifyNodeConfiguration holds the configuration for the InotifyNode.
type InotifyNodeConfiguration struct {
	RuleChainID string   `json:"ruleChainId"`
	StartNodeID string   `json:"startNodeId,omitempty"`
	Description string   `json:"description,omitempty"`
	Path        string   `json:"path"`
	Events      []string `json:"events"`
}

// InotifyNode is a component that actively listens for file system events.
type InotifyNode struct {
	types.BaseNode
	types.Instance
	nodeConfig  InotifyNodeConfiguration
	runtimePool types.RuntimePool
	watcher     *fsnotify.Watcher
	logger      types.Logger
	cancel      context.CancelFunc
}

// New creates a new instance of the node for the NodeManager.
func (n *InotifyNode) New() types.Node {
	return &InotifyNode{
		BaseNode: n.BaseNode, // Explicitly reference the prototype's BaseNode
		logger:   log.GetLogger(),
	}
}

// Init initializes the node with its static configuration.
func (n *InotifyNode) Init(config types.Config) error {
	if err := utils.Decode(config, &n.nodeConfig); err != nil {
		return fmt.Errorf("failed to decode inotify node config: %w", err)
	}
	if n.nodeConfig.Path == "" {
		return fmt.Errorf("inotify node config 'path' is required")
	}
	if n.nodeConfig.RuleChainID == "" {
		return fmt.Errorf("inotify node config 'ruleChainId' is required")
	}
	return nil
}

// SetRuntimePool implements the types.Endpoint interface.
func (n *InotifyNode) SetRuntimePool(pool any) error {
	if p, ok := pool.(types.RuntimePool); ok {
		n.runtimePool = p
		return nil
	}
	return fmt.Errorf("provided pool is not of type types.RuntimePool")
}

// Start begins the active listening process for file system events.
func (n *InotifyNode) Start(ctx context.Context) error {
	var err error
	n.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	if err := n.watcher.Add(n.nodeConfig.Path); err != nil {
		n.watcher.Close()
		return fmt.Errorf("failed to add path '%s' to watcher: %w", n.nodeConfig.Path, err)
	}

	ctx, n.cancel = context.WithCancel(ctx)

	go func() {
		defer n.watcher.Close()
		for {
			select {
			case <-ctx.Done():
				n.logger.Infof(ctx, "Stopping inotify watcher for path: %s", n.nodeConfig.Path)
				return
			case event, ok := <-n.watcher.Events:
				if !ok {
					return
				}
				n.handleEvent(ctx, event)
			case err, ok := <-n.watcher.Errors:
				if !ok {
					return
				}
				n.logger.Errorf(ctx, "Inotify watcher error: %v", err)
			}
		}
	}()

	n.logger.Infof(ctx, "Started inotify watcher for path: %s", n.nodeConfig.Path)
	return nil
}

// Stop terminates the listening process.
func (n *InotifyNode) Stop() error {
	if n.cancel != nil {
		n.cancel()
	}
	return nil
}

// Destroy cleans up resources used by the node.
func (n *InotifyNode) Destroy() {
	n.Stop()
}

// GetInstance implements the types.SharedNode interface, returning the node itself.
func (n *InotifyNode) GetInstance() (interface{}, error) {
	return n, nil
}

// GetConfiguration returns the node's configuration for inspection.
func (n *InotifyNode) Configuration() InotifyNodeConfiguration {
	return n.nodeConfig
}

// handleEvent is the internal method that processes the file system event.
func (n *InotifyNode) handleEvent(ctx context.Context, event fsnotify.Event) {
	if !n.shouldHandleEvent(event) {
		return
	}

	content, err := os.ReadFile(event.Name)
	if err != nil {
		n.logger.Errorf(ctx, "InotifyNode failed to read file %s: %v", event.Name, err)
		return
	}

	if len(content) == 0 {
		return
	}

	metadata := make(types.Metadata)
	metadata["source"] = "inotify"
	metadata["path"] = event.Name
	metadata["event"] = event.Op.String()
	metadata["filename"] = filepath.Base(event.Name)

	msg := types.NewMsg(n.nodeConfig.RuleChainID, "", metadata, nil).WithDataFormat(types.TEXT)
	msg.SetData(string(content))

	onEnd := func(msg types.RuleMsg, err error) {
		if err != nil {
			n.logger.Errorf(ctx, "Inotify-triggered chain execution failed. chainId=%s, error=%v", n.nodeConfig.RuleChainID, err)
		}
	}

	var rt types.Runtime
	var ok bool

	if n.runtimePool != nil {
		rt, ok = n.runtimePool.Get(n.nodeConfig.RuleChainID)
	} else {
		// Fallback to global default runtime pool if none is injected.
		rt, ok = registry.Default.RuntimePool.Get(n.nodeConfig.RuleChainID)
	}

	if !ok {
		n.logger.Errorf(ctx, "runtime not found for rule chain: %s", n.nodeConfig.RuleChainID)
		return
	}

	// Execute starting from the specified startNodeId, or from the root if not specified.
	err = rt.Execute(ctx, n.nodeConfig.StartNodeID, msg, onEnd)
	if err != nil {
		n.logger.Errorf(ctx, "InotifyNode failed to start rule chain execution %s: %v", n.nodeConfig.RuleChainID, err)
	}
}

func (n *InotifyNode) shouldHandleEvent(event fsnotify.Event) bool {
	if len(n.nodeConfig.Events) == 0 {
		return true
	}
	for _, e := range n.nodeConfig.Events {
		if strings.EqualFold(e, event.Op.String()) {
			return true
		}
	}
	return false
}
