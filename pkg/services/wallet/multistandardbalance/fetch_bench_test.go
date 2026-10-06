package multistandardbalance

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/status-im/go-wallet-sdk/pkg/balance/multistandardfetcher"
	"github.com/status-im/go-wallet-sdk/pkg/contracts/multicall3"

	"github.com/status-im/status-go/internal/rpc/chain/ethclient"
	"github.com/status-im/status-go/pkg/services/wallet/puzzleauth"
)

// The fake node runs in a child process so its allocations and CPU time stay out
// of the measured fetch.
const fakeRPCEnv = "MULTISTANDARDBALANCE_FAKE_RPC"

const (
	fakeChainID        = uint64(1)
	fakeTokensPerChain = 8500
	fakeAccounts       = 2
)

var fakeMulticallAddress = common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")

func fakeTokenAddress(i int) common.Address {
	var a common.Address
	a[0] = 0x70
	binary.BigEndian.PutUint32(a[16:], uint32(i))
	return a
}

func fakeAccountAddress(i int) common.Address {
	var a common.Address
	a[0] = 0xac
	a[19] = byte(i + 1)
	return a
}

// fakeBalance is the node's answer for balanceOf(account) on the token: most
// balances are zero, as on a fresh profile; failed reports a reverted sub-call.
func fakeBalance(account, token common.Address) (balance *big.Int, failed bool) {
	idx := int(binary.BigEndian.Uint32(token[16:]))
	switch {
	case idx%1009 == 7:
		return nil, true
	case idx%97 == 3:
		b := new(big.Int).Mul(big.NewInt(int64(idx+1)), big.NewInt(1_000_000_000_000_000))
		return b.Add(b, big.NewInt(int64(account[19]))), false
	default:
		return new(big.Int), false
	}
}

func fakeExpectedERC20Balances(account common.Address) (nonZero map[common.Address]*big.Int, failed map[common.Address]bool) {
	nonZero = make(map[common.Address]*big.Int)
	failed = make(map[common.Address]bool)
	for i := 0; i < fakeTokensPerChain; i++ {
		token := fakeTokenAddress(i)
		balance, fail := fakeBalance(account, token)
		if fail {
			failed[token] = true
		} else if balance.Sign() != 0 {
			nonZero[token] = balance
		}
	}
	return nonZero, failed
}

func fakeTokenAddresses() []common.Address {
	tokens := make([]common.Address, fakeTokensPerChain)
	for i := range tokens {
		tokens[i] = fakeTokenAddress(i)
	}
	return tokens
}

func fakeMulticallResponse(data []byte) ([]byte, error) {
	mcABI, err := multicall3.Multicall3MetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	method, err := mcABI.MethodById(data[:4])
	if err != nil {
		return nil, err
	}
	args, err := method.Inputs.Unpack(data[4:])
	if err != nil {
		return nil, err
	}
	calls := *abi.ConvertType(args[1], new([]multicall3.IMulticall3Call)).(*[]multicall3.IMulticall3Call)
	results := make([]multicall3.IMulticall3Result, len(calls))
	for i, call := range calls {
		if call.Target == (common.Address{19: 0x64}) {
			results[i] = multicall3.IMulticall3Result{Success: false, ReturnData: []byte{}}
			continue
		}
		balance, failed := fakeBalance(common.BytesToAddress(call.CallData[4:36]), call.Target)
		if failed {
			results[i] = multicall3.IMulticall3Result{Success: false, ReturnData: []byte{}}
			continue
		}
		results[i] = multicall3.IMulticall3Result{Success: true, ReturnData: common.LeftPadBytes(balance.Bytes(), 32)}
	}
	switch method.Name {
	case "tryBlockAndAggregate":
		return method.Outputs.Pack(big.NewInt(21_000_000), common.Hash{1, 2, 3}, results)
	case "tryAggregate":
		return method.Outputs.Pack(results)
	}
	return nil, fmt.Errorf("unexpected method %s", method.Name)
}

func serveFakeMulticallRPC(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID     json.RawMessage   `json:"id"`
		Method string            `json:"method"`
		Params []json.RawMessage `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	reply := map[string]any{"jsonrpc": "2.0", "id": req.ID}
	var msg struct {
		Data  hexutil.Bytes `json:"data"`
		Input hexutil.Bytes `json:"input"`
	}
	if req.Method != "eth_call" || len(req.Params) == 0 || json.Unmarshal(req.Params[0], &msg) != nil {
		reply["error"] = map[string]any{"code": -32601, "message": "unsupported " + req.Method}
	} else {
		data := msg.Input
		if len(data) == 0 {
			data = msg.Data
		}
		out, err := fakeMulticallResponse(data)
		if err != nil {
			reply["error"] = map[string]any{"code": -32000, "message": err.Error()}
		} else {
			reply["result"] = hexutil.Bytes(out)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(reply)
}

// TestFakeMulticallRPCProcess is the child process entry point; it is a no-op in
// a regular test run.
func TestFakeMulticallRPCProcess(t *testing.T) {
	if os.Getenv(fakeRPCEnv) != "1" {
		t.Skip("fake RPC child process only")
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := &http.Server{Handler: http.HandlerFunc(serveFakeMulticallRPC), ReadHeaderTimeout: time.Minute}
	go func() { _ = server.Serve(ln) }()
	fmt.Printf("FAKE_RPC_URL=http://%s\n", ln.Addr())
	_, _ = io.Copy(io.Discard, os.Stdin)
}

func startFakeMulticallRPC(tb testing.TB) string {
	cmd := exec.Command(os.Args[0], "-test.run=^TestFakeMulticallRPCProcess$", "-test.v") //nolint:gosec // this test binary
	cmd.Env = append(os.Environ(), fakeRPCEnv+"=1")
	stdin, err := cmd.StdinPipe()
	require.NoError(tb, err)
	stdout, err := cmd.StdoutPipe()
	require.NoError(tb, err)
	require.NoError(tb, cmd.Start())
	tb.Cleanup(func() {
		_ = stdin.Close()
		_ = cmd.Wait()
	})
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		if url, ok := strings.CutPrefix(scanner.Text(), "FAKE_RPC_URL="); ok {
			go func() { _, _ = io.Copy(io.Discard, stdout) }()
			return url
		}
	}
	tb.Fatal("fake RPC process did not start")
	return ""
}

type fakeEthClientGetter struct {
	client ethclient.EthClientInterface
}

func (g *fakeEthClientGetter) EthClient(uint64) (ethclient.EthClientInterface, error) {
	return g.client, nil
}

type noopLastBlockManager struct{}

func (noopLastBlockManager) SetLatestBlockNumber(uint64, uint64) {}

// newFakeFetchController wires the production fetch path (status-go RPC client
// behind the puzzleauth transport, Fetcher, storage) against the fake node.
func newFakeFetchController(tb testing.TB) (*Controller, multistandardfetcher.FetchConfig) {
	url := startFakeMulticallRPC(tb)
	httpClient := &http.Client{Timeout: time.Minute, Transport: puzzleauth.NewTransport(url, nil)}
	rpcClient, err := rpc.DialOptions(context.Background(), url, rpc.WithHTTPClient(httpClient))
	require.NoError(tb, err)
	tb.Cleanup(rpcClient.Close)

	fetcher := NewFetcher(&fakeEthClientGetter{client: ethclient.NewEthClient(rpcClient)}, DefaultBatchSize,
		map[uint64]common.Address{fakeChainID: fakeMulticallAddress})
	c := NewController(DefaultControllerConfig(), NewStorageMemory(), fetcher, nil, nil, nil, nil, nil,
		noopLastBlockManager{}, nil, zap.NewNop())

	config := multistandardfetcher.FetchConfig{ERC20: make(map[AccountAddress][]ContractAddress), OmitZeroERC20Balances: true}
	tokens := fakeTokenAddresses()
	for i := 0; i < fakeAccounts; i++ {
		config.ERC20[fakeAccountAddress(i)] = tokens
	}
	return c, config
}

func runFakeFetch(tb testing.TB, c *Controller, config multistandardfetcher.FetchConfig) {
	ctx := context.Background()
	resultsCh, err := c.fetcher.FetchBalances(ctx, fakeChainID, config)
	require.NoError(tb, err)
	for result := range resultsCh {
		c.handleFetchResult(ctx, fakeChainID, result)
	}
}

func cpuTime() time.Duration {
	var ru syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &ru)
	return time.Duration(ru.Utime.Nano() + ru.Stime.Nano())
}

func TestFetchStoresNonZeroERC20Balances(t *testing.T) {
	c, config := newFakeFetchController(t)
	runFakeFetch(t, c, config)

	for account := range config.ERC20 {
		balances, state, err := c.storage.GetERC20Balances(context.Background(), BalancesKey{Account: account, ChainID: fakeChainID})
		require.NoError(t, err)
		require.NotEqual(t, NeverFetched, state.FetchedAt)
		require.Equal(t, int64(21_000_000), state.AtBlockNumber.Int64())

		nonZero, failed := fakeExpectedERC20Balances(account)
		require.NotEmpty(t, nonZero)
		require.NotEmpty(t, failed)
		// Zero balances are not stored; the unanswered tokens are stored as unknown (nil).
		require.Len(t, balances, len(nonZero)+len(failed))
		for token, expected := range nonZero {
			require.Zero(t, expected.Cmp(balances[token]), token.Hex())
		}
		for token := range failed {
			value, present := balances[token]
			require.True(t, present, token.Hex())
			require.Nil(t, value, token.Hex())
		}
	}
}

// BenchmarkFetchERC20Balances measures one full ERC20 balance fetch (2 accounts x
// 8500 tokens = 17000 calls, 7 requests) as the controller runs it, from building
// the calls to storing the result. cpu-ms/op is this process only.
func BenchmarkFetchERC20Balances(b *testing.B) {
	c, config := newFakeFetchController(b)
	runFakeFetch(b, c, config) // warm up connections and caches

	b.ReportAllocs()
	b.ResetTimer()
	start := cpuTime()
	for i := 0; i < b.N; i++ {
		runFakeFetch(b, c, config)
	}
	b.ReportMetric(float64((cpuTime()-start).Microseconds())/1000/float64(b.N), "cpu-ms/op")
}
