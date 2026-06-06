package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/neohetj/matrix/pkg/types"
)

const EndpointNodeType = "endpoint/mcp"

// LoadEndpointsFromDir loads endpoint/mcp JSON node definitions from a module
// endpoint directory.
func LoadEndpointsFromDir(dir string, opts ...Option) ([]*Endpoint, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, fmt.Errorf("endpoint directory is required")
	}
	stat, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("stat endpoint directory: %w", err)
	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("endpoint path %q is not a directory", dir)
	}

	var endpoints []*Endpoint
	err = filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			return nil
		}
		endpoint, err := LoadEndpointFromFile(path, opts...)
		if err != nil {
			return err
		}
		if endpoint != nil {
			endpoints = append(endpoints, endpoint)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("no endpoint/mcp node definitions found under %s", dir)
	}
	return endpoints, nil
}

// LoadEndpointFromFile loads one endpoint/mcp JSON node definition. Non-MCP
// endpoint files are ignored and return nil.
func LoadEndpointFromFile(path string, opts ...Option) (*Endpoint, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read MCP endpoint %s: %w", path, err)
	}
	var def types.NodeDef
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("decode MCP endpoint %s: %w", path, err)
	}
	if strings.TrimSpace(def.Type) != EndpointNodeType {
		return nil, nil
	}
	cfg, err := decodeEndpointConfig(def.Configuration)
	if err != nil {
		return nil, fmt.Errorf("decode MCP endpoint configuration %s: %w", path, err)
	}
	return NewEndpoint(cfg, opts...)
}

func decodeEndpointConfig(config types.ConfigMap) (types.McpEndpointNodeConfiguration, error) {
	var cfg types.McpEndpointNodeConfiguration
	data, err := json.Marshal(config)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
