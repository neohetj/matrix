package mcp

import (
	"fmt"
	"strings"

	"github.com/neohetj/matrix/pkg/types"
)

// NewTextToolResult wraps plain text as a single MCP text content item.
func NewTextToolResult(text string) types.McpToolResult {
	return textToolResult(text)
}

// NewErrorToolResult wraps sanitized text as an MCP tool error.
func NewErrorToolResult(text string) types.McpToolResult {
	return errorToolResult(text)
}

// SanitizeToolText redacts bearer tokens and common secret-looking key/value
// pairs before text leaves the MCP adapter.
func SanitizeToolText(text string) string {
	return sanitizeToolText(text)
}

// NewHTTPToolResult normalizes captured HTTP response data into a single text
// MCP result, applying size limits and secret redaction consistently across
// transport-backed and in-process targets.
func NewHTTPToolResult(statusCode int, body []byte, maxBytes int64) types.McpToolResult {
	limit := maxBytes
	if limit <= 0 {
		limit = 1 << 20
	}
	data := body
	truncated := int64(len(data)) > limit
	if truncated {
		data = data[:limit]
	}
	text := strings.TrimSpace(string(data))
	if truncated {
		text += "\n[truncated]"
	}
	if text == "" {
		text = fmt.Sprintf("HTTP %d", statusCode)
	}
	text = sanitizeToolText(text)
	result := textToolResult(text)
	if statusCode >= 400 {
		result.IsError = true
	}
	return result
}
