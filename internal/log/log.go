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

// Package log provides a logging facade for the Matrix engine.
// It offers a set of helper functions to log messages with contextual information
// automatically extracted from the NodeCtx.
package log

import (
	"context"
	"sync"

	"github.com/neohetj/matrix/pkg/types"
)

const (
	LevelDebug = "debug"
	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"
)

var (
	// globalLogger is the default logger for the entire engine.
	// It can be replaced by a host application using SetLogger.
	globalLogger types.Logger = &NoopLogger{}
	// mu protects globalLogger during concurrent SetLogger calls.
	mu sync.RWMutex
)

// SetLogger sets the global logger for the matrix engine.
// This function should be called only once at application startup.
func SetLogger(logger types.Logger) {
	mu.Lock()
	defer mu.Unlock()
	globalLogger = logger
}

// GetLogger returns the currently configured global logger.
func GetLogger() types.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return globalLogger
}

// NoopLogger is a no-op logger to prevent panics when no logger is configured.
type NoopLogger struct{}

func (n *NoopLogger) Printf(ctx context.Context, format string, v ...any) {}
func (n *NoopLogger) Debugf(ctx context.Context, format string, v ...any) {}
func (n *NoopLogger) Infof(ctx context.Context, format string, v ...any)  {}
func (n *NoopLogger) Warnf(ctx context.Context, format string, v ...any)  {}
func (n *NoopLogger) Errorf(ctx context.Context, format string, v ...any) {}
func (n *NoopLogger) With(fields ...any) types.Logger                     { return n }

// getEffectiveLogger determines the correct logger to use based on the context.
// It prioritizes the instance-specific logger and falls back to the global logger.
func getEffectiveLogger(ctx types.NodeCtx) types.Logger {
	// Check if the runtime context has a specific logger.
	if ctx != nil && ctx.Logger() != nil {
		return ctx.Logger()
	}
	// Fallback to the global logger.
	return GetLogger()
}

// logWithFields is a private helper function that extracts context, merges fields,
// and returns a prepared logger instance.
func logWithFields(ctx types.NodeCtx, fields ...any) types.Logger {
	logger := getEffectiveLogger(ctx)

	// 1. Extract base fields from the context.
	var baseFields []any
	if ctx != nil {
		if chainID := ctx.ChainID(); chainID != "" {
			baseFields = append(baseFields, "chainId", chainID)
		}
		if nodeID := ctx.NodeID(); nodeID != "" {
			baseFields = append(baseFields, "nodeId", nodeID)
		}
	}

	// 2. Merge base fields with the provided business fields.
	allFields := append(baseFields, fields...)

	// 3. Use With to attach all fields and return.
	return logger.With(allFields...)
}

// Debug logs a message at Debug level.
func Debug(ctx types.NodeCtx, msg string, fields ...any) {
	logWithFields(ctx, fields...).Debugf(ctx.GetContext(), msg)
}

// Info logs a message at Info level.
func Info(ctx types.NodeCtx, msg string, fields ...any) {
	logWithFields(ctx, fields...).Infof(ctx.GetContext(), msg)
}

// Warn logs a message at Warn level.
func Warn(ctx types.NodeCtx, msg string, fields ...any) {
	logWithFields(ctx, fields...).Warnf(ctx.GetContext(), msg)
}

// Error logs a message at Error level.
func Error(ctx types.NodeCtx, msg string, fields ...any) {
	logWithFields(ctx, fields...).Errorf(ctx.GetContext(), msg)
}
