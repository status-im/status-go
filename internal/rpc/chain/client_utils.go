package chain

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/internal/requestgzip"
	"github.com/status-im/status-go/internal/traffic"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/services/wallet/puzzleauth"
)

// rpcTransport is shared by every RPC client so that which hosts take gzip
// request bodies is learnt once per host; privateRPCTransport is the same for
// the providers users added.
var (
	rpcTransport        = requestgzip.New(traffic.Transport)
	privateRPCTransport = requestgzip.New(traffic.PrivateTransport)
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
	case params.PuzzleAuth:
		origin, err := puzzleauth.OriginForURL(provider.URL.Reveal())
		if err != nil {
			return nil, fmt.Errorf("puzzle auth: invalid provider URL for %s: %w", provider.Name, err)
		}
		opts = append(opts, rpc.WithHTTPClient(puzzleauth.NewHTTPClient(origin, rpcTransport)))
	default:
		return nil, fmt.Errorf("unknown auth type: %s", provider.AuthType)
	}
	if provider.AuthType != params.PuzzleAuth {
		// A provider the user added is theirs: its host stays out of the report.
		transport := rpcTransport
		if provider.Type == params.UserProviderType {
			transport = privateRPCTransport
		}
		opts = append(opts, rpc.WithHTTPClient(&http.Client{Transport: transport}))
	}

	opts = append(opts, rpc.WithHeaders(headers))

	// Dial the RPC client
	rpcClient, err := rpc.DialOptions(context.Background(), provider.URL.Reveal(), opts...)
	if err != nil {
		return nil, fmt.Errorf("dial server failed for provider %s: %w", provider.Name, err)
	}

	return rpcClient, nil
}
