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
	"context"
	"fmt"
	"sync"
	"time"

	"embed"

	"github.com/neohetj/matrix/internal/registry"

	"github.com/neohetj/matrix/internal/aop"
	"github.com/neohetj/matrix/internal/builder"
	_ "github.com/neohetj/matrix/internal/builtin"
	"github.com/neohetj/matrix/internal/endpointgate"
	"github.com/neohetj/matrix/internal/log"
	"github.com/neohetj/matrix/internal/runtime"
	"github.com/neohetj/matrix/pkg/config"
	"github.com/neohetj/matrix/pkg/facotry"
	"github.com/neohetj/matrix/pkg/trace"
	"github.com/neohetj/matrix/pkg/types"

	_ "github.com/neohetj/matrix/pkg/components/action"
	_ "github.com/neohetj/matrix/pkg/components/external"
	_ "github.com/neohetj/matrix/pkg/components/ops"
)

func init() {
	types.NewNodeCtx = facotry.NewNodeCtx
	types.NewMsg = facotry.NewMsg
	types.CloneMsgWithDataT = facotry.CloneMsgWithDataT
	types.NewDataT = facotry.NewDataT
	types.NewSubMsg = facotry.NewSubMsg
	types.NewCoreObj = facotry.NewCoreObj
	types.NewCoreObjDef = facotry.NewCoreObjDef
}

// Registry is the default, global instance of the registry.
var Registry = types.DefaultRegistry

// --- Public Helper Functions ---

// Discover is a public API wrapper around the internal builder function.
func Discover(
	dslLoader types.ResourceProvider,
	loaderCfg config.LoaderConfig,
	enabledComponents []string,
) (rulechainPaths []string, endpointPaths []string, sharedNodePaths []string) {
	return builder.DiscoverComponentPaths(dslLoader, loaderCfg, enabledComponents)
}

// SetLogger sets the global logger for the entire matrix engine.
// This should be called by the host application at startup.
func SetLogger(logger types.Logger) {
	log.SetLogger(logger)
}

// MatrixEngine is the facade for the refactored rule engine.
// It holds an internal reference to the registry and exposes its components via methods.
type MatrixEngine struct {
	config                config.MatrixConfig
	registry              types.RegistryProvider
	traceManager          *trace.Manager
	loader                types.ResourceProvider
	logger                types.Logger
	embedFSs              []embed.FS
	activeCancel          context.CancelFunc
	activeWG              sync.WaitGroup
	moduleConfigs         map[string]types.ConfigReader
	nodeConfigOwners      map[string]string
	ruleChainConfigOwners map[string]string
	defaultConfigModule   string
	moduleConfigErr       error
	configActiveEndpoints []types.ActiveEndpoint
}

// --- Getters for core components ---

func (e *MatrixEngine) RuntimePool() types.RuntimePool {
	if e.registry == nil {
		return nil
	}
	return e.registry.GetRuntimePool()
}

func (e *MatrixEngine) SharedNodePool() types.NodePool {
	if e.registry == nil {
		return nil
	}
	return e.registry.GetSharedNodePool()
}

func (e *MatrixEngine) NodeManager() types.NodeManager {
	if e.registry == nil {
		return nil
	}
	return e.registry.GetNodeManager()
}

func (e *MatrixEngine) NodeFuncManager() types.NodeFuncManager {
	if e.registry == nil {
		return nil
	}
	return e.registry.GetNodeFuncManager()
}

// GetEngineConfig retrieves a value from the global business configuration.
func (e *MatrixEngine) GetEngineConfig(key string) (any, bool) {
	return e.config.GetEngineConfig(key)
}

// TraceManager returns the trace manager.
// Note: This method is not part of the types.MatrixEngine interface.
func (e *MatrixEngine) TraceManager() types.SnapshotFinalizer {
	if e.traceManager == nil {
		return nil
	}
	return e.traceManager
}

// Config returns the full matrix configuration.
// Note: This method is not part of the types.MatrixEngine interface.
func (e *MatrixEngine) Config() config.MatrixConfig { return e.config }

func (e *MatrixEngine) Loader() types.ResourceProvider { return e.loader }
func (e *MatrixEngine) BizConfig() types.ConfigMap     { return e.config.Business }
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

// WithRegistry sets a custom registry provider for the engine.
func WithRegistry(r types.RegistryProvider) Option {
	return func(e *MatrixEngine) {
		e.registry = r
	}
}

// --- Constructors ---

// New is the simplified entry point for the Matrix engine.
// It initializes, configures, and builds a complete engine instance.
func New(cfg config.MatrixConfig, opts ...Option) (*MatrixEngine, error) {
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
	if engine.moduleConfigErr != nil {
		return nil, engine.moduleConfigErr
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
	rulechainPaths, endpointPaths, sharedNodePaths := e.discoverComponents()

	sch, err := e.initScheduler()
	if err != nil {
		return nil, err
	}
	built := false
	defer func() {
		if !built && len(e.moduleConfigs) > 0 {
			e.abortModuleConfigStartup()
			sch.Stop()
		}
	}()

	defs, err := e.loadDefinitions(rulechainPaths)
	if err != nil {
		return nil, err
	}

	if err := e.initRegistryAndLoadComponents(sharedNodePaths, endpointPaths); err != nil {
		return nil, err
	}

	runtimeOpts := e.initRuntimeOpts()
	if err := e.initRuntimes(sch, defs, runtimeOpts); err != nil {
		return nil, err
	}
	if err := e.StartActiveEndpoints(context.Background()); err != nil {
		return nil, err
	}

	built = true
	return e, nil
}

// StartActiveEndpoints starts all loaded endpoints that actively listen for external events.
func (e *MatrixEngine) StartActiveEndpoints(ctx context.Context) error {
	if e.registry == nil || e.activeCancel != nil {
		return nil
	}
	endpointCtx, cancel := context.WithCancel(ctx)
	e.activeCancel = cancel
	for _, endpoint := range e.registry.GetSharedNodePool().GetEndpoints() {
		active, ok := endpoint.(types.ActiveEndpoint)
		if !ok {
			continue
		}
		enabled, err := endpointgate.Enabled(endpoint, e.config.Business)
		if err != nil {
			cancel()
			return fmt.Errorf("active endpoint %s: %w", endpoint.ID(), err)
		}
		if !enabled {
			if e.logger != nil {
				e.logger.Infof(endpointCtx, "active endpoint %s disabled by configuration, skipped", endpoint.ID())
			}
			continue
		}
		if len(e.moduleConfigs) > 0 {
			// Start 可能部分建立监听后失败，因此先登记到本实例的撤销清单。
			e.configActiveEndpoints = append(e.configActiveEndpoints, active)
		}
		if err := active.Start(endpointCtx); err != nil {
			cancel()
			if _, aware := endpoint.(types.ConfigReaderAware); aware {
				// 配置消费者的驱动错误可能含连接凭据，不向宿主传播原始 cause。
				err = types.ErrConfigInitialization
			}
			return fmt.Errorf("failed to start active endpoint %s: %w", endpoint.ID(), err)
		}
	}
	return nil
}

// StopActiveEndpoints stops all active endpoints started by the engine.
func (e *MatrixEngine) StopActiveEndpoints() error {
	if e.activeCancel != nil {
		e.activeCancel()
		e.activeCancel = nil
	}
	if e.registry == nil {
		return nil
	}
	var stopErr error
	for _, endpoint := range e.registry.GetSharedNodePool().GetEndpoints() {
		active, ok := endpoint.(types.ActiveEndpoint)
		if !ok {
			continue
		}
		if err := active.Stop(); err != nil && stopErr == nil {
			stopErr = err
		}
	}
	return stopErr
}

func (e *MatrixEngine) discoverComponents() (rulechainPaths, endpointPaths, sharedNodePaths []string) {
	return Discover(e.loader, e.config.Loader, e.config.EnabledComponents)
}

func (e *MatrixEngine) initScheduler() (types.Scheduler, error) {
	if e.config.Scheduler.Type == "" {
		e.config.Scheduler.Type = "ants"
	}
	sch, err := builder.NewSchedulerFromConfig(e.config.Scheduler)
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduler: %w", err)
	}
	return sch, nil
}

func (e *MatrixEngine) loadDefinitions(rulechainPaths []string) (map[string]*types.RuleChainDef, error) {
	defs, err := builder.LoadDefs(e.loader, rulechainPaths)
	if err != nil {
		return nil, fmt.Errorf("failed to load rule chain definitions: %w", err)
	}
	return defs, nil
}

func (e *MatrixEngine) initRegistryAndLoadComponents(sharedNodePaths, endpointPaths []string) error {
	if e.registry == nil {
		e.registry = registry.Default
	}
	if err := e.prepareModuleConfig(); err != nil {
		return err
	}

	nodeMgr := e.registry.GetNodeManager()
	sharedNodePool := e.registry.GetSharedNodePool()
	pool := e.registry.GetRuntimePool()

	if err := builder.LoadSharedNodes(e.loader, sharedNodePaths, nodeMgr, sharedNodePool); err != nil {
		return fmt.Errorf("failed to load shared nodes: %w", err)
	}

	if err := builder.LoadEndpoints(e.loader, endpointPaths, nodeMgr, sharedNodePool, pool); err != nil {
		return fmt.Errorf("failed to load endpoints: %w", err)
	}
	return nil
}

func (e *MatrixEngine) initRuntimeOpts() []runtime.Option {
	var runtimeOpts []runtime.Option

	if e.config.Trace.EnableAop {
		traceStore := trace.NewInMemoryStore(24 * time.Hour)
		e.traceManager = trace.NewManager(traceStore)
		tracer := e.traceManager.GetTracer()
		traceAspect := aop.NewTraceAspect(tracer)
		runtimeOpts = append(runtimeOpts, runtime.WithAspects(traceAspect))
	}

	runtimeOpts = append(runtimeOpts, runtime.WithLogger(e.logger))
	return runtimeOpts
}

func (e *MatrixEngine) initRuntimes(sch types.Scheduler, defs map[string]*types.RuleChainDef, runtimeOpts []runtime.Option) error {
	merger := builder.NewMerger(defs)
	pool := e.registry.GetRuntimePool()

	for id := range defs {
		finalDef, err := merger.Merge(id)
		if err != nil {
			return fmt.Errorf("failed to merge rule chain %s: %w", id, err)
		}

		rtOpts := append(runtimeOpts, runtime.WithEngine(e))
		rt, err := runtime.NewDefaultRuntime(sch, finalDef, rtOpts...)
		if err != nil {
			return fmt.Errorf("failed to create runtime for chain %s: %w", id, err)
		}
		if err := pool.Register(id, rt); err != nil {
			return fmt.Errorf("failed to register runtime for chain %s: %w", id, err)
		}
	}
	return nil
}
