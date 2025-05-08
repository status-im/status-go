package chain

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/params"
)

// CreateEthClientFromProvider creates an Ethereum RPC client from the given RpcProvider.
func CreateEthClientFromProvider(provider params.RpcProvider, rpcUserAgentName string) (*rpc.Client, error) {
	if !provider.Enabled {
		return nil, nil
	}

	// Create RPC client options
	var opts []rpc.ClientOption
	headers := http.Header{}
	headers.Set("User-Agent", rpcUserAgentName)

	// Set up authentication if needed
	switch provider.AuthType {
	case params.BasicAuth:
		authEncoded := base64.StdEncoding.EncodeToString([]byte(provider.AuthLogin.Append(":", provider.AuthPassword).Reveal()))
		headers.Set("Authorization", "Basic "+authEncoded)
	case params.TokenAuth:
		provider.URL = provider.URL.Append(provider.AuthToken)
	case params.NoAuth:
		// no-op
	default:
		return nil, fmt.Errorf("unknown auth type: %s", provider.AuthType)
	}

	opts = append(opts, rpc.WithHeaders(headers))

	// Dial the RPC client
	rpcClient, err := rpc.DialOptions(context.Background(), provider.URL.Reveal(), opts...)
	if err != nil {
		return nil, fmt.Errorf("dial server failed for provider %s: %w", provider.Name, err)
	}

	return rpcClient, nil
}
