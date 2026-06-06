package mcp

import (
	"context"
	"net/http"
	"net/textproto"
)

type incomingHTTPHeadersContextKey struct{}

// WithIncomingHTTPHeaders copies HTTP request headers into the context so MCP
// auth resolvers can consume trusted gateway assertions without reaching back
// into the transport layer.
func WithIncomingHTTPHeaders(ctx context.Context, headers http.Header) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	normalized := http.Header{}
	for key, values := range headers {
		canonicalKey := textproto.CanonicalMIMEHeaderKey(key)
		normalized[canonicalKey] = append([]string(nil), values...)
	}
	return context.WithValue(ctx, incomingHTTPHeadersContextKey{}, normalized)
}

// IncomingHTTPHeadersFromContext returns the copied request headers previously
// attached by WithIncomingHTTPHeaders.
func IncomingHTTPHeadersFromContext(ctx context.Context) (http.Header, bool) {
	if ctx == nil {
		return nil, false
	}
	headers, ok := ctx.Value(incomingHTTPHeadersContextKey{}).(http.Header)
	if !ok || headers == nil {
		return nil, false
	}
	return headers.Clone(), true
}
