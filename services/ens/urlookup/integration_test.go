package urlookup_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/go-wallet-sdk/pkg/ethclient"

	"github.com/status-im/status-go/internal/contracts/universalresolver"
	"github.com/status-im/status-go/services/ens/ccipread"
	"github.com/status-im/status-go/services/ens/urlookup"
)

// Integration tests against the canonical ENSv2 sentinels documented at
// https://docs.ens.domains/web/ensv2-readiness/. They are skipped unless
// ENS_INTEGRATION=1 is set; ENS_RPC_URL must point to an Ethereum mainnet
// RPC. Run with:
//
//	ENS_INTEGRATION=1 ENS_RPC_URL=https://... go test ./services/ens/urlookup -run Integration

const (
	sentinelURName     = "ur.integration-tests.eth"
	sentinelURAddress  = "0x2222222222222222222222222222222222222222"
	sentinelCCIPName   = "test.offchaindemo.eth"
	sentinelCCIPAddrss = "0x779981590E7Ccc0CFAe8040Ce7151324747cDb97"
)

func mustClient(t *testing.T) *ethclient.Client {
	t.Helper()
	if os.Getenv("ENS_INTEGRATION") != "1" {
		t.Skip("set ENS_INTEGRATION=1 to run ENS integration tests")
	}
	url := os.Getenv("ENS_RPC_URL")
	if url == "" {
		t.Fatal("ENS_RPC_URL must be set when ENS_INTEGRATION=1 (Ethereum mainnet RPC)")
	}
	dialCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	rpcClient, err := rpc.DialContext(dialCtx, url)
	if err != nil {
		t.Fatalf("dial %s: %v", url, err)
	}
	t.Cleanup(rpcClient.Close)
	return ethclient.NewClient(rpcClient)
}

// TestIntegration_UR_AddressSentinel proves the Universal Resolver is
// reachable and ABI-compatible. ur.integration-tests.eth is documented to
// resolve to the all-2s address; any other result means the library / UR
// binding is stale.
func TestIntegration_UR_AddressSentinel(t *testing.T) {
	ec := mustClient(t)
	backend := ccipread.New(ec)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	addr, err := urlookup.Address(ctx, backend, universalresolver.CanonicalAddress, sentinelURName)
	if err != nil {
		t.Fatalf("Address(%s): %v", sentinelURName, err)
	}
	if !strings.EqualFold(addr.Hex(), sentinelURAddress) {
		t.Fatalf("got %s, want %s", addr.Hex(), sentinelURAddress)
	}
}

// TestIntegration_UR_OffchainSentinel proves CCIP-Read works end-to-end.
// test.offchaindemo.eth lives behind an offchain gateway, so resolving it
// exercises the OffchainLookup revert path and the HTTPS gateway round-trip.
func TestIntegration_UR_OffchainSentinel(t *testing.T) {
	ec := mustClient(t)
	backend := ccipread.New(ec)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	addr, err := urlookup.Address(ctx, backend, universalresolver.CanonicalAddress, sentinelCCIPName)
	if err != nil {
		t.Fatalf("Address(%s): %v", sentinelCCIPName, err)
	}
	if !strings.EqualFold(addr.Hex(), sentinelCCIPAddrss) {
		t.Fatalf("got %s, want %s", addr.Hex(), sentinelCCIPAddrss)
	}
}
