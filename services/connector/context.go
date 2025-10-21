package connector

import (
	"context"
)

// ConnectionType represents the source of the RPC connection
type ConnectionType string

const (
	ConnectionTypeHTTP     ConnectionType = "http"     // Untrusted - from WebSocket
	ConnectionTypeInternal ConnectionType = "internal" // Trusted - from CallRPC (status-desktop)
)

// ContextKey is a type used for keys in connector Context.
type ContextKey struct {
	Name string
}

var (
	connectionTypeKey = ContextKey{Name: "connectionType"}
)

// WithConnectionType adds connection type to context
func WithConnectionType(ctx context.Context, connType ConnectionType) context.Context {
	return context.WithValue(ctx, connectionTypeKey, connType)
}

// GetConnectionType retrieves connection type from context
func GetConnectionType(ctx context.Context) ConnectionType {
	if connType, ok := ctx.Value(connectionTypeKey).(ConnectionType); ok {
		return connType
	}
	return ConnectionTypeInternal // default to trusted
}

// IsUntrustedConnection checks if the connection is from HTTP/WebSocket
func IsUntrustedConnection(ctx context.Context) bool {
	return GetConnectionType(ctx) == ConnectionTypeHTTP
}
