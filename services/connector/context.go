package connector

import (
	"context"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
)

// ConnectionType represents trust level of the RPC connection
type ConnectionType string

const (
	ConnectionTypeTrusted   ConnectionType = "trusted"   // IPC or explicitly set as trusted
	ConnectionTypeUntrusted ConnectionType = "untrusted" // HTTP, WebSocket, or unknown
)

// ContextKey is a type used for keys in connector Context.
type ContextKey struct {
	Name string
}

var (
	connectionTypeKey = ContextKey{Name: "connectionType"}
	requestOriginKey  = ContextKey{Name: "requestOrigin"}
)

// WithConnectionType adds connection type to context
func WithConnectionType(ctx context.Context, connType ConnectionType) context.Context {
	return context.WithValue(ctx, connectionTypeKey, connType)
}

// WithRequestOrigin stores the HTTP Origin from an untrusted browser connection
// (WebSocket / HTTP). Used to bind dApp identity instead of trusting JSON "url".
func WithRequestOrigin(ctx context.Context, origin string) context.Context {
	return context.WithValue(ctx, requestOriginKey, origin)
}

// GetRequestOrigin returns the browser Origin if present in context.
func GetRequestOrigin(ctx context.Context) string {
	if origin, ok := ctx.Value(requestOriginKey).(string); ok {
		return origin
	}
	return ""
}

// GetConnectionType retrieves connection type from context
func GetConnectionType(ctx context.Context) ConnectionType {
	// Check go-ethereum's PeerInfo first - this is set automatically by the RPC server
	peerInfo := gethrpc.PeerInfoFromContext(ctx)
	if peerInfo.Transport == "ipc" {
		return ConnectionTypeTrusted
	}

	// Check custom context value (explicitly set connection type)
	if connType, ok := ctx.Value(connectionTypeKey).(ConnectionType); ok {
		return connType
	}

	// Default to untrusted for safety
	return ConnectionTypeUntrusted
}

// IsUntrustedConnection checks if the connection should not be trusted
func IsUntrustedConnection(ctx context.Context) bool {
	return GetConnectionType(ctx) == ConnectionTypeUntrusted
}
