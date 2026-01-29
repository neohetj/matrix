// Package builder provides internal helper functions for constructing the MatrixEngine.
package builder

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/neohetj/matrix/internal/loader"
	"github.com/neohetj/matrix/internal/log"
	"github.com/neohetj/matrix/internal/parser"
	"github.com/neohetj/matrix/internal/scheduler"

	"github.com/neohetj/matrix/pkg/config"
	"github.com/neohetj/matrix/pkg/types"
)

type DefMap map[string]*types.RuleChainDef

// NewSchedulerFromConfig creates a scheduler instance from its configuration.
func NewSchedulerFromConfig(cfg config.SchedulerConfig) (types.Scheduler, error) {
	switch strings.ToLower(cfg.Type) {
	case "ants":
		return scheduler.NewAntsScheduler(cfg.PoolSize)
	default:
		return nil, fmt.Errorf("unknown scheduler type: %s", cfg.Type)
	}
}

// NewLoaderFromConfig creates a resource loader from its configuration.
func NewLoaderFromConfig(cfg config.LoaderConfig, logger types.Logger, embedFSs ...embed.FS) (types.ResourceProvider, error) {
	if logger == nil {
		logger = log.GetLogger()
	}
	hybridLoader := loader.NewHybridLoader(logger)
	embedIndex := 0
	for _, p := range cfg.Providers {
		switch strings.ToLower(p.Type) {
		case "file":
			priority := 50
			if p.Priority != nil {
				priority = *p.Priority
			}
			if priority < 0 || priority > 100 {
				fmt.Printf("warning: 'Priority' for file provider is out of range (0-100) and will be set to default. Got: %d\n", priority)
			}
			if len(p.Args) == 0 {
				return nil, fmt.Errorf("file provider requires at least one base path argument")
			}
			for _, path := range p.Args {
				hybridLoader.AddProvider(loader.NewFileProvider(path, priority))
			}
		case "embed":
			if p.Priority != nil {
				fmt.Println("warning: 'Priority' is not applicable to 'embed' providers and will be ignored.")
			}
			if len(p.Args) > 0 {
				fmt.Println("warning: 'Args' are not applicable to 'embed' providers and will be ignored.")
			}
			if embedIndex >= len(embedFSs) {
				return nil, fmt.Errorf("embed provider configured but no embed.FS provided")
			}
			hybridLoader.AddProvider(loader.NewEmbedProvider(embedFSs[embedIndex]))
			embedIndex++
		default:
			return nil, fmt.Errorf("unknown loader provider type: %s", p.Type)
		}
	}
	return hybridLoader, nil
}

// LoadDefs scans for rule chain definitions from a list of base paths.
func LoadDefs(dslLoader types.ResourceProvider, rulechainPaths []string) (DefMap, error) {
	chains := make(DefMap)
	jsonParser := &parser.JsonParser{}

	for _, basePath := range rulechainPaths {
		err := dslLoader.WalkDir(basePath, func(filePath string, d fs.DirEntry, err error) error {
			if err != nil {
				// If a component's rulechains dir doesn't exist, it's not an error.
				if errors.Is(err, fs.ErrNotExist) {
					return filepath.SkipDir
				}
				return err
			}
			if d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
				return nil
			}

			res, err := dslLoader.ReadFile(filePath)
			if err != nil {
				fmt.Printf("warning: failed to read rule chain file %s: %v\n", filePath, err)
				return nil
			}

			chainDef, err := jsonParser.DecodeRuleChain(res.Content)
			if err != nil {
				fmt.Printf("warning: failed to decode rule chain %s (source: %s): %v\n", filePath, res.Source, err)
				return nil
			}

			if chainDef.RuleChain.ID == "" {
				chainDef.RuleChain.ID = strings.TrimSuffix(d.Name(), ".json")
			}

			if _, exists := chains[chainDef.RuleChain.ID]; exists {
				return fmt.Errorf("duplicate rule chain ID found: %s from %s", chainDef.RuleChain.ID, res.Source)
			}
			chains[chainDef.RuleChain.ID] = chainDef
			return nil
		})

		if err != nil && err != filepath.SkipDir {
			return nil, fmt.Errorf("error walking for rule chains in '%s': %w", basePath, err)
		}
	}
	return chains, nil
}

// LoadEndpoints scans for endpoint definitions from a list of base paths.
func LoadEndpoints(
	dslLoader types.ResourceProvider,
	endpointPaths []string,
	nodeMgr types.NodeManager,
	nodePool types.NodePool,
	runtimePool types.RuntimePool,
) error {
	for _, basePath := range endpointPaths {
		err := dslLoader.WalkDir(basePath, func(filePath string, d fs.DirEntry, err error) error {
			if err != nil {
				// If a component's endpoints dir doesn't exist, it's not an error.
				if errors.Is(err, fs.ErrNotExist) {
					return filepath.SkipDir
				}
				return err
			}
			if d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
				return nil
			}

			res, err := dslLoader.ReadFile(filePath)
			if err != nil {
				fmt.Printf("warning: failed to read endpoint file %s: %v\n", filePath, err)
				return nil
			}

			jsonParser := &parser.JsonParser{}
			nodeDef, err := jsonParser.DecodeNode(res.Content)
			if err != nil {
				fmt.Printf("warning: failed to decode endpoint file %s (source: %s): %v\n", filePath, res.Source, err)
				return nil
			}

			ctx, err := nodePool.NewFromNodeDef(*nodeDef, nodeMgr)
			if err != nil {
				fmt.Printf("warning: failed to load endpoint from def %s: %v\n", nodeDef.ID, err)
				return nil
			}

			// If the created node is an endpoint, inject the runtime pool.
			if endpoint, ok := ctx.GetNode().(types.Endpoint); ok {
				if err := endpoint.SetRuntimePool(runtimePool); err != nil {
					fmt.Printf("warning: failed to set runtime pool for endpoint %s: %v\n", nodeDef.ID, err)
					return nil // Or handle as a non-fatal error
				}
			}
			return nil
		})

		if err != nil && err != filepath.SkipDir {
			return fmt.Errorf("error walking for endpoints in '%s': %w", basePath, err)
		}
	}
	return nil
}

// LoadSharedNodes scans for shared node definitions from a list of base paths.
func LoadSharedNodes(
	dslLoader types.ResourceProvider,
	sharedNodePaths []string,
	nodeMgr types.NodeManager,
	nodePool types.NodePool,
) error {
	for _, basePath := range sharedNodePaths {
		err := dslLoader.WalkDir(basePath, func(filePath string, d fs.DirEntry, err error) error {
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return filepath.SkipDir
				}
				return err
			}
			if d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
				return nil
			}

			res, err := dslLoader.ReadFile(filePath)
			if err != nil {
				fmt.Printf("warning: failed to read shared node file %s: %v\n", filePath, err)
				return nil
			}

			jsonParser := &parser.JsonParser{}
			def, err := jsonParser.DecodeRuleChain(res.Content)
			if err != nil {
				fmt.Printf("warning: failed to decode shared node file %s (source: %s): %v\n", filePath, res.Source, err)
				return nil
			}

			// A shared node file is a rulechain def used as a container for nodes.
			if _, err := nodePool.LoadFromRuleChainDef(def, nodeMgr); err != nil {
				fmt.Printf("warning: failed to load shared nodes from %s: %v\n", filePath, err)
				return nil
			}
			return nil
		})

		if err != nil && err != filepath.SkipDir {
			return fmt.Errorf("error walking for shared nodes in '%s': %w", basePath, err)
		}
	}
	return nil
}

// Merger handles the merging of rule chain definitions based on `imports`.
type Merger struct {
	defs DefMap
}

// NewMerger creates a new merger instance.
func NewMerger(defs DefMap) *Merger {
	return &Merger{defs: defs}
}

// Merge performs the merge operation for a given root definition ID.
func (m *Merger) Merge(rootID string) (*types.RuleChainDef, error) {
	visited := make(map[string]bool)
	return m.mergeRecursive(rootID, visited)
}

func (m *Merger) mergeRecursive(defID string, visited map[string]bool) (*types.RuleChainDef, error) {
	if visited[defID] {
		return nil, fmt.Errorf("circular import detected: %s", defID)
	}
	visited[defID] = true

	currentDef, ok := m.defs[defID]
	if !ok {
		return nil, fmt.Errorf("definition with id '%s' not found", defID)
	}

	currentDef = deepCopyDef(currentDef)

	if len(currentDef.RuleChain.Attrs.Imports) == 0 {
		return currentDef, nil
	}

	baseDef := &types.RuleChainDef{}
	for _, importID := range currentDef.RuleChain.Attrs.Imports {
		importedDef, err := m.mergeRecursive(importID, visited)
		if err != nil {
			return nil, fmt.Errorf("failed to process import %s in %s: %w", importID, defID, err)
		}
		baseDef = mergeDefs(baseDef, importedDef)
	}

	finalDef := mergeDefs(baseDef, currentDef)
	finalDef.RuleChain = currentDef.RuleChain
	finalDef.Metadata.Connections = currentDef.Metadata.Connections

	return finalDef, nil
}

func mergeDefs(base, overlay *types.RuleChainDef) *types.RuleChainDef {
	if base == nil {
		return overlay
	}
	if overlay == nil {
		return base
	}

	merged := &types.RuleChainDef{}
	merged.RuleChain = overlay.RuleChain

	nodeMap := make(map[string]types.NodeDef)
	for _, node := range base.Metadata.Nodes {
		nodeMap[node.ID] = node
	}
	for _, node := range overlay.Metadata.Nodes {
		nodeMap[node.ID] = node
	}
	for _, node := range nodeMap {
		merged.Metadata.Nodes = append(merged.Metadata.Nodes, node)
	}

	merged.Metadata.Relations = append(merged.Metadata.Relations, base.Metadata.Relations...)
	merged.Metadata.Relations = append(merged.Metadata.Relations, overlay.Metadata.Relations...)
	merged.Metadata.Connections = overlay.Metadata.Connections

	return merged
}

func deepCopyDef(def *types.RuleChainDef) *types.RuleChainDef {
	tempParser := parser.NewJsonParser()
	bytes, err := tempParser.EncodeRuleChain(def)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal RuleChainDef for deep copy: %v", err))
	}
	newDef, err := tempParser.DecodeRuleChain(bytes)
	if err != nil {
		panic(fmt.Sprintf("failed to unmarshal RuleChainDef for deep copy: %v", err))
	}
	return newDef
}

// DiscoverComponentPaths scans a root directory for component subdirectories using the provided loader
// and returns paths to their 'rulechains', 'endpoints', and 'shared' directories if they exist.
func DiscoverComponentPaths(
	dslLoader types.ResourceProvider,
	componentsRoot string,
	enabledComponents []string,
) (rulechainPaths []string, endpointPaths []string, sharedNodePaths []string) {
	componentSet := make(map[string]struct{})
	componentSet["common"] = struct{}{} // "common" is always included
	for _, name := range enabledComponents {
		componentSet[name] = struct{}{}
	}

	for name := range componentSet {
		rulechainPath := filepath.ToSlash(filepath.Join(componentsRoot, name, "dsl/rulechains"))
		if _, err := dslLoader.Stat(rulechainPath); err == nil {
			rulechainPaths = append(rulechainPaths, rulechainPath)
		}

		endpointPath := filepath.ToSlash(filepath.Join(componentsRoot, name, "dsl/endpoints"))
		if _, err := dslLoader.Stat(endpointPath); err == nil {
			endpointPaths = append(endpointPaths, endpointPath)
		}

		sharedPath := filepath.ToSlash(filepath.Join(componentsRoot, name, "dsl/shared"))
		if _, err := dslLoader.Stat(sharedPath); err == nil {
			sharedNodePaths = append(sharedNodePaths, sharedPath)
		}
	}

	return
}
