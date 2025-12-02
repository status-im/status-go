package connector

import (
	"context"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
)

// ConnectionType represents the source of the RPC connection
type ConnectionType string

const (
	ConnectionTypeHTTP     ConnectionType = "http"     // Untrusted - from HTTP
	ConnectionTypeWS       ConnectionType = "ws"       // Untrusted - from WebSocket
	ConnectionTypeInternal ConnectionType = "internal" // Trusted - from internal RPC (status-desktop)
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
	// Check go-ethereum's PeerInfo first - this is set automatically by the RPC server
	// for HTTP and WebSocket connections
	peerInfo := gethrpc.PeerInfoFromContext(ctx)
	if peerInfo.Transport == "ws" {
		return ConnectionTypeWS
	}
	if peerInfo.Transport == "http" {
		return ConnectionTypeHTTP
	}

	// Fall back to context property
	if connType, ok := ctx.Value(connectionTypeKey).(ConnectionType); ok {
		return connType
	}
	return ConnectionTypeInternal // default to untrusted
}

// IsUntrustedConnection checks if the connection is from HTTP/WebSocket
func IsUntrustedConnection(ctx context.Context) bool {
	connType := GetConnectionType(ctx)
	return connType == ConnectionTypeHTTP || connType == ConnectionTypeWS
}
