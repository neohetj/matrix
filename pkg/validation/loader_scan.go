package validation

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/neohetj/matrix/pkg/types"
)

func scanRuleChainPaths(provider types.ResourceProvider, paths []string, report *Report, ruleChainIDs map[string]struct{}, defs *[]*types.RuleChainDef) {
	for _, basePath := range paths {
		walkJSON(provider, basePath, report, TargetRuleChain, func(filePath string, content []byte) {
			var def types.RuleChainDef
			if err := json.Unmarshal(content, &def); err != nil {
				addLoaderFailure(report, filePath, fmt.Errorf("decode rulechain: %w", err))
				return
			}
			if def.RuleChain.ID == "" {
				def.RuleChain.ID = strings.TrimSuffix(filepath.Base(filePath), ".json")
			}
			setSourcePath(def.Metadata.Nodes, filePath)
			ruleChainIDs[def.RuleChain.ID] = struct{}{}
			*defs = append(*defs, &def)
		})
	}
}

func scanSharedPaths(provider types.ResourceProvider, paths []string, report *Report, sharedIDs map[string]struct{}) {
	for _, basePath := range paths {
		walkJSON(provider, basePath, report, TargetSharedRef, func(filePath string, content []byte) {
			var def types.RuleChainDef
			if err := json.Unmarshal(content, &def); err != nil {
				addLoaderFailure(report, filePath, fmt.Errorf("decode shared nodes: %w", err))
				return
			}
			for _, node := range def.Metadata.Nodes {
				if strings.TrimSpace(node.ID) != "" {
					sharedIDs[node.ID] = struct{}{}
				}
			}
		})
	}
}

func scanEndpointPaths(provider types.ResourceProvider, paths []string, report *Report, endpoints *[]*types.NodeDef) {
	for _, basePath := range paths {
		walkJSON(provider, basePath, report, TargetEndpoint, func(filePath string, content []byte) {
			var def types.NodeDef
			if err := json.Unmarshal(content, &def); err != nil {
				addLoaderFailure(report, filePath, fmt.Errorf("decode endpoint: %w", err))
				return
			}
			def.SourcePath = filePath
			*endpoints = append(*endpoints, &def)
		})
	}
}

func walkJSON(provider types.ResourceProvider, basePath string, report *Report, targetKind TargetKind, handle func(filePath string, content []byte)) {
	basePath = filepath.ToSlash(strings.TrimSpace(basePath))
	if basePath == "" {
		return
	}
	err := provider.WalkDir(basePath, func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return filepath.SkipDir
			}
			report.AddIssue(Issue{
				Code:     CodeLoaderFailure,
				Severity: SeverityError,
				Message:  err.Error(),
				Target: Target{
					Kind: targetKind,
					Path: filepath.ToSlash(filePath),
				},
			})
			return nil
		}
		if d == nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}
		filePath = filepath.ToSlash(filePath)
		res, readErr := provider.ReadFile(filePath)
		if readErr != nil {
			addLoaderFailure(report, filePath, fmt.Errorf("read file: %w", readErr))
			return nil
		}
		handle(filePath, res.Content)
		return nil
	})
	if err != nil && err != filepath.SkipDir {
		report.AddIssue(Issue{
			Code:     CodeLoaderFailure,
			Severity: SeverityError,
			Message:  err.Error(),
			Target: Target{
				Kind: targetKind,
				Path: basePath,
			},
		})
	}
}

func setSourcePath(nodes []types.NodeDef, sourcePath string) {
	for i := range nodes {
		nodes[i].SourcePath = sourcePath
	}
}

func addLoaderFailure(report *Report, filePath string, err error) {
	report.AddIssue(Issue{
		Code:     CodeLoaderFailure,
		Severity: SeverityError,
		Message:  err.Error(),
		Target: Target{
			Kind:       TargetLoader,
			Path:       filepath.ToSlash(filePath),
			SourcePath: filepath.ToSlash(filePath),
		},
	})
}
