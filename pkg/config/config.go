// Package config provides the configuration structures for the Matrix engine.
package config

// LoaderProviderConfig defines the configuration for a single resource provider.
type LoaderProviderConfig struct {
	Type     string   `json:"Type"`               // "file" or "embed"
	Args     []string `json:"Args,omitempty"`     // Arguments for the provider (e.g., base path for "file")
	Priority *int     `json:"Priority,omitempty"` // Search priority, higher numbers are checked first.
}

// LoaderConfig defines the configuration for the DSL loader.
// It supports a list of providers to be composed into a HybridLoader.
type LoaderConfig struct {
	Providers      []LoaderProviderConfig `json:"Providers"`
	EndpointsPath  string                 `json:"EndpointsPath"`
	ComponentsRoot string                 `json:"ComponentsRoot,omitempty"` // New field
}

// SchedulerConfig defines the configuration for the task scheduler.
type SchedulerConfig struct {
	// Type of the scheduler, e.g., "ants".
	Type string `json:"Type"`
	// PoolSize is the number of goroutines in the scheduler's pool.
	PoolSize int `json:"PoolSize"`
}

// TraceConfig defines the configuration for the tracing system.
type TraceConfig struct {
	// EnableAop determines whether the trace aspect is automatically applied to all runtimes.
	EnableAop bool `json:"EnableAop"`
}

// Config is the main configuration object for the Matrix engine.
// It can be unmarshaled from a YAML file (via json tags).
type Config struct {
	Loader            LoaderConfig    `json:"Loader"`
	Scheduler         SchedulerConfig `json:"Scheduler"`
	Trace             TraceConfig     `json:"Trace"`
	EnabledComponents []string        `json:"EnabledComponents"`
}

// NewConfig creates a new Config object with sensible defaults.
func NewConfig() Config {
	var cfg Config

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
