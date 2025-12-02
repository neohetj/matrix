/*
 * Copyright 2025 The Matrix Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package matrix is the main entry point for the Matrix rule engine.
// It provides a facade for initializing and managing all engine components.
package matrix

import (
	"fmt"
	"time"

	"embed"

	"gitlab.com/neohet/matrix/internal/aop"
	"gitlab.com/neohet/matrix/internal/builder"
	"gitlab.com/neohet/matrix/internal/log"
	"gitlab.com/neohet/matrix/internal/runtime"
	"gitlab.com/neohet/matrix/pkg/config"
	"gitlab.com/neohet/matrix/pkg/registry"
	"gitlab.com/neohet/matrix/pkg/trace"
	"gitlab.com/neohet/matrix/pkg/types"

	_ "gitlab.com/neohet/matrix/pkg/components/action"
	_ "gitlab.com/neohet/matrix/pkg/components/endpoint"
	_ "gitlab.com/neohet/matrix/pkg/components/external"
	_ "gitlab.com/neohet/matrix/pkg/components/functions"
	_ "gitlab.com/neohet/matrix/pkg/components/link"
	_ "gitlab.com/neohet/matrix/pkg/components/ops"
)

// --- Public Helper Functions ---

// Discover is a public API wrapper around the internal builder function.
func Discover(
	dslLoader types.ResourceProvider,
	componentsRoot string,
	enabledComponents []string,
) (rulechainPaths []string, endpointPaths []string, sharedNodePaths []string) {
	return builder.DiscoverComponentPaths(dslLoader, componentsRoot, enabledComponents)
}

// SetLogger sets the global logger for the entire matrix engine.
// This should be called by the host application at startup.
func SetLogger(logger types.Logger) {
	log.SetLogger(logger)
}

// MatrixEngine is the facade for the refactored rule engine.
// It holds an internal reference to the registry and exposes its components via methods.
type MatrixEngine struct {
	config       config.Config
	registry     types.RegistryProvider
	traceManager *trace.Manager
	loader       types.ResourceProvider
	logger       types.Logger
	embedFSs     []embed.FS
}

// --- Getters for core components ---

func (e *MatrixEngine) RuntimePool() types.RuntimePool { return e.registry.GetRuntimePool() }
func (e *MatrixEngine) SharedNodePool() types.NodePool { return e.registry.GetSharedNodePool() }
func (e *MatrixEngine) NodeManager() types.NodeManager { return e.registry.GetNodeManager() }
func (e *MatrixEngine) NodeFuncManager() types.NodeFuncManager {
	return e.registry.GetNodeFuncManager()
}
func (e *MatrixEngine) TraceManager() *trace.Manager   { return e.traceManager }
func (e *MatrixEngine) Loader() types.ResourceProvider { return e.loader }
func (e *MatrixEngine) Config() config.Config          { return e.config }
func (e *MatrixEngine) Logger() types.Logger           { return e.logger }

// Option is a function that configures the MatrixEngine.
type Option func(*MatrixEngine)

// WithLoader sets a custom resource loader for the engine.
func WithLoader(l types.ResourceProvider) Option {
	return func(e *MatrixEngine) {
		e.loader = l
	}
}

// WithLogger sets a custom logger for this specific engine instance.
// If this option is not used, the engine will default to using the
// global logger configured via matrix.SetLogger().
func WithLogger(l types.Logger) Option {
	return func(e *MatrixEngine) {
		e.logger = l
	}
}

// WithEmbedFS adds an embed.FS as a resource provider to the engine.
// It will be added alongside any providers defined in the config.
func WithEmbedFS(fs embed.FS) Option {
	return func(e *MatrixEngine) {
		e.embedFSs = append(e.embedFSs, fs)
	}
}

// --- Constructors ---

// New is the simplified entry point for the Matrix engine.
// It initializes, configures, and builds a complete engine instance.
func New(cfg config.Config, opts ...Option) (*MatrixEngine, error) {
	engine := &MatrixEngine{
		config: cfg,
	}

	// Set a default logger immediately to prevent nil pointers in options.
	// The WithLogger option can override this default.
	engine.logger = log.GetLogger()

	// Apply functional options to configure the engine instance.
	for _, opt := range opts {
		opt(engine)
	}

	// If a loader wasn't provided via an option, create a default one from the config.
	if engine.loader == nil {
		var err error
		engine.loader, err = builder.NewLoaderFromConfig(cfg.Loader, engine.logger, engine.embedFSs...)
		if err != nil {
			return nil, fmt.Errorf("failed to create loader from config: %w", err)
		}
	}

	// Call the private constructor to finalize the build.
	return newEngine(engine)
}

// newEngine is the underlying, private constructor for the Matrix engine.
// It takes a pre-configured engine instance and the discovered component paths to finalize the build.
func newEngine(e *MatrixEngine) (*MatrixEngine, error) {
	componentsRoot := e.config.Loader.ComponentsRoot
	if componentsRoot == "" {
		componentsRoot = "components" // Default convention
	}
	rulechainPaths, endpointPaths, sharedNodePaths := Discover(e.loader, componentsRoot, e.config.EnabledComponents)

	// Defensive coding: If scheduler type is not set, default to "ants".
	if e.config.Scheduler.Type == "" {
		e.config.Scheduler.Type = "ants"
	}

	sch, err := builder.NewSchedulerFromConfig(e.config.Scheduler)
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduler: %w", err)
	}

	defs, err := builder.LoadDefs(e.loader, rulechainPaths)
	if err != nil {
		return nil, fmt.Errorf("failed to load rule chain definitions: %w", err)
	}

	reg := registry.Default
	nodeMgr := reg.GetNodeManager()
	sharedNodePool := reg.GetSharedNodePool()
	pool := reg.GetRuntimePool()

	if err := builder.LoadSharedNodes(e.loader, sharedNodePaths, nodeMgr, sharedNodePool); err != nil {
		return nil, fmt.Errorf("failed to load shared nodes: %w", err)
	}

	if err := builder.LoadEndpoints(e.loader, endpointPaths, nodeMgr, sharedNodePool, pool); err != nil {
		return nil, fmt.Errorf("failed to load endpoints: %w", err)
	}
	merger := builder.NewMerger(defs)

	var runtimeOpts []runtime.Option
	var traceManager *trace.Manager

	if e.config.Trace.EnableAop {
		// Create a default in-memory trace store.
		// TODO: Make the store type configurable.
		traceStore := trace.NewInMemoryStore(24 * time.Hour)
		traceManager = trace.NewManager(traceStore)
		tracer := traceManager.GetTracer()
		traceAspect := aop.NewTraceAspect(tracer)
		runtimeOpts = append(runtimeOpts, runtime.WithAspects(traceAspect))
	}

	runtimeOpts = append(runtimeOpts, runtime.WithLogger(e.logger))
	for id, _ := range defs {
		finalDef, err := merger.Merge(id)
		if err != nil {
			return nil, fmt.Errorf("failed to merge rule chain %s: %w", id, err)
		}
		rt, err := runtime.NewDefaultRuntime(sch, finalDef, runtimeOpts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create runtime for chain %s: %w", id, err)
		}
		if err := pool.Register(id, rt); err != nil {
			return nil, fmt.Errorf("failed to register runtime for chain %s: %w", id, err)
		}
	}

	e.registry = reg
	e.traceManager = traceManager

	return e, nil
}
