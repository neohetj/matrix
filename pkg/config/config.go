// Package config provides the configuration structures for the Matrix engine.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/neohetj/matrix/pkg/types"
)

// LoadFromFile loads the configuration from a YAML file.
func LoadFromFile(path string) (MatrixConfig, error) {
	var cfg MatrixConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cfg, nil
}

// LoaderProviderConfig defines the configuration for a single resource provider.
type LoaderProviderConfig struct {
	Type     string   `json:"Type" yaml:"type"`                             // "file" or "embed"
	Args     []string `json:"Args,omitempty" yaml:"args,omitempty"`         // Arguments for the provider (e.g., base path for "file")
	Priority *int     `json:"Priority,omitempty" yaml:"priority,omitempty"` // Search priority, higher numbers are checked first.
}

// LoaderConfig defines the configuration for the DSL loader.
// It supports a list of providers to be composed into a HybridLoader.
type LoaderConfig struct {
	Providers      []LoaderProviderConfig `json:"Providers" yaml:"providers"`
	EndpointsPath  string                 `json:"EndpointsPath" yaml:"endpointsPath"`
	ComponentsRoot string                 `json:"ComponentsRoot,omitempty" yaml:"componentsRoot,omitempty"`
}

// SchedulerConfig defines the configuration for the task scheduler.
type SchedulerConfig struct {
	// Type of the scheduler, e.g., "ants".
	Type string `json:"Type" yaml:"type"`
	// PoolSize is the number of goroutines in the scheduler's pool.
	PoolSize int `json:"PoolSize" yaml:"poolSize"`
}

// TraceConfig defines the configuration for the tracing system.
type TraceConfig struct {
	// EnableAop determines whether the trace aspect is automatically applied to all runtimes.
	EnableAop bool `json:"EnableAop" yaml:"enableAop"`
}

// MatrixConfig is the main configuration object for the Matrix engine.
// It can be unmarshaled from a YAML file (via json tags).
type MatrixConfig struct {
	Loader            LoaderConfig    `json:"Loader" yaml:"loader"`
	Scheduler         SchedulerConfig `json:"Scheduler" yaml:"scheduler"`
	Trace             TraceConfig     `json:"Trace" yaml:"trace"`
	EnabledComponents []string        `json:"EnabledComponents" yaml:"enabledComponents"`
	Business          types.ConfigMap `json:"Business" yaml:"business"`
}

// GetEngineConfig retrieves a value from the Business config map.
// If the key is not found in the map, it attempts to retrieve it from environment variables.
// The key is converted to uppercase when looking up environment variables.
func (c *MatrixConfig) GetEngineConfig(key string) (any, bool) {
	if c.Business != nil {
		if val, ok := c.Business[key]; ok {
			return val, true
		}
	}
	// Fallback to Env
	return os.LookupEnv(key)
}

// NewConfig creates a new Config object with sensible defaults.
func NewConfig() MatrixConfig {
	var cfg MatrixConfig

	cfg.Loader.Providers = []LoaderProviderConfig{
		{
			Type: "file",
			Args: []string{"."}, // Default to current directory
		},
	}

	cfg.Scheduler.Type = "ants"
	cfg.Scheduler.PoolSize = 100

	cfg.Trace.EnableAop = false

	return cfg
}
