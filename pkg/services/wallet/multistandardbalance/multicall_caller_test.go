package multistandardbalance

import (
	"context"
	"encoding/json"
	"io"
	"math/big"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/status-im/go-wallet-sdk/pkg/contracts/multicall3"
	"github.com/status-im/go-wallet-sdk/pkg/multicall"

	"github.com/status-im/status-go/internal/rpc/chain/ethclient"
)

func multicall3ABI(t testing.TB) *abi.ABI {
	a, err := multicall3.Multicall3MetaData.GetAbi()
	require.NoError(t, err)
	return a
}

func newTestRand(seed int64) *rand.Rand {
	return rand.New(rand.NewSource(seed)) // #nosec G404 -- reproducible test data
}

func randomAddress(r *rand.Rand) common.Address {
	var a common.Address
	r.Read(a[:])
	return a
}

func randomBytes(r *rand.Rand, n int) []byte {
	b := make([]byte, n)
	r.Read(b)
	return b
}

// testCalls mixes balanceOf calls with the other call shapes a fetch sends
// (arbBlockNumber, getEthBalance, ownerOf-like) and odd calldata lengths.
func testCalls(r *rand.Rand, n int) []multicall3.IMulticall3Call {
	calls := make([]multicall3.IMulticall3Call, 0, n)
	for i := 0; i < n; i++ {
		switch i % 7 {
		case 3:
			calls = append(calls, multicall.BuildChainBlockNumberCall())
		case 4:
			calls = append(calls, multicall3.IMulticall3Call{Target: randomAddress(r), CallData: randomBytes(r, 68)})
		case 5:
			calls = append(calls, multicall3.IMulticall3Call{Target: randomAddress(r), CallData: randomBytes(r, r.Intn(100))})
		case 6:
			calls = append(calls, multicall3.IMulticall3Call{Target: randomAddress(r)})
		default:
			calls = append(calls, multicall.BuildERC20BalanceCall(randomAddress(r), randomAddress(r)))
		}
	}
	return calls
}

func TestEncodeMulticallInputMatchesABIPack(t *testing.T) {
	mcABI := multicall3ABI(t)
	r := newTestRand(1)
	for _, method := range []string{"tryAggregate", "tryBlockAndAggregate"} {
		for _, size := range []int{0, 1, 2, 7, 64, 499, 2500} {
			for _, requireSuccess := range []bool{false, true} {
				calls := testCalls(r, size)
				expected, err := mcABI.Pack(method, requireSuccess, calls)
				require.NoError(t, err)
				require.Equal(t, expected, encodeMulticallInput(mcABI.Methods[method].ID, requireSuccess, calls),
					"%s size=%d requireSuccess=%v", method, size, requireSuccess)
			}
		}
	}
}

func TestEncodeMulticallInputBalanceOfOnly(t *testing.T) {
	mcABI := multicall3ABI(t)
	r := newTestRand(2)
	for _, size := range []int{1, 2499, 2500} {
		account := randomAddress(r)
		calls := make([]multicall3.IMulticall3Call, size)
		for i := range calls {
			calls[i] = multicall.BuildERC20BalanceCall(account, randomAddress(r))
		}
		expected, err := mcABI.Pack("tryBlockAndAggregate", false, calls)
		require.NoError(t, err)
		require.Equal(t, expected, encodeMulticallInput(mcABI.Methods["tryBlockAndAggregate"].ID, false, calls))
	}
}

func testResults(r *rand.Rand, n int) []multicall3.IMulticall3Result {
	results := make([]multicall3.IMulticall3Result, n)
	for i := range results {
		switch i % 5 {
		case 0:
			results[i] = multicall3.IMulticall3Result{Success: true, ReturnData: make([]byte, 32)}
		case 1:
			results[i] = multicall3.IMulticall3Result{Success: true, ReturnData: common.LeftPadBytes(big.NewInt(r.Int63()).Bytes(), 32)}
		case 2:
			results[i] = multicall3.IMulticall3Result{Success: false, ReturnData: []byte{}}
		case 3:
			results[i] = multicall3.IMulticall3Result{Success: false, ReturnData: randomBytes(r, r.Intn(100))}
		default:
			results[i] = multicall3.IMulticall3Result{Success: true, ReturnData: []byte{}}
		}
	}
	return results
}

func TestDecodeMulticallOutputMatchesABIUnpack(t *testing.T) {
	mcABI := multicall3ABI(t)
	r := newTestRand(3)
	for _, size := range []int{0, 1, 2, 9, 2500} {
		results := testResults(r, size)

		out, err := mcABI.Methods["tryAggregate"].Outputs.Pack(results)
		require.NoError(t, err)
		decoded, err := decodeTryAggregateOutput(out)
		require.NoError(t, err)
		require.Equal(t, normalizeResults(results), normalizeResults(decoded))

		blockNumber := new(big.Int).SetUint64(r.Uint64())
		blockHash := common.BytesToHash(randomBytes(r, 32))
		out, err = mcABI.Methods["tryBlockAndAggregate"].Outputs.Pack(blockNumber, blockHash, results)
		require.NoError(t, err)
		bn, bh, decoded, err := decodeTryBlockAndAggregateOutput(out)
		require.NoError(t, err)
		require.Zero(t, blockNumber.Cmp(bn))
		require.Equal(t, [32]byte(blockHash), bh)
		require.Equal(t, normalizeResults(results), normalizeResults(decoded))
	}
}

// normalizeResults maps nil and empty return data to the same value, as abi.Unpack does.
func normalizeResults(results []multicall3.IMulticall3Result) []multicall3.IMulticall3Result {
	out := make([]multicall3.IMulticall3Result, len(results))
	for i, res := range results {
		out[i] = multicall3.IMulticall3Result{Success: res.Success, ReturnData: append([]byte{}, res.ReturnData...)}
	}
	return out
}

func TestDecodeMulticallOutputRejectsMalformed(t *testing.T) {
	mcABI := multicall3ABI(t)
	r := newTestRand(4)
	valid, err := mcABI.Methods["tryBlockAndAggregate"].Outputs.Pack(big.NewInt(7), common.Hash{}, testResults(r, 6))
	require.NoError(t, err)

	word := func(v uint64) []byte { return common.LeftPadBytes(new(big.Int).SetUint64(v).Bytes(), 32) }
	setWord := func(b []byte, at int, v []byte) []byte {
		c := append([]byte{}, b...)
		copy(c[at:at+32], v)
		return c
	}
	huge := common.LeftPadBytes(new(big.Int).Lsh(big.NewInt(1), 255).Bytes(), 32)

	cases := map[string][]byte{
		"empty":                   nil,
		"truncated head":          valid[:64],
		"truncated tail":          valid[:len(valid)-1],
		"array offset past end":   setWord(valid, 64, word(uint64(len(valid)))),
		"array offset huge":       setWord(valid, 64, huge),
		"array length huge":       setWord(valid, 96, huge),
		"array length past end":   setWord(valid, 96, word(1000)),
		"element offset past end": setWord(valid, 128, word(uint64(len(valid)))),
		"element offset huge":     setWord(valid, 128, huge),
	}
	// First element: success word, bytes offset word, bytes length word.
	first := 128 + int(new(big.Int).SetBytes(valid[128:160]).Uint64())
	cases["bool not 0 or 1"] = setWord(valid, first, word(2))
	cases["bytes offset huge"] = setWord(valid, first+32, huge)
	cases["bytes length past end"] = setWord(valid, first+64, word(uint64(len(valid))))
	cases["bytes length huge"] = setWord(valid, first+64, huge)

	for name, out := range cases {
		t.Run(name, func(t *testing.T) {
			_, _, _, err := decodeTryBlockAndAggregateOutput(out)
			require.Error(t, err)
			_, abiErr := mcABI.Unpack("tryBlockAndAggregate", out)
			require.Error(t, abiErr, "abi.Unpack accepts it too, the case is not malformed")
		})
	}
}

func FuzzDecodeTryAggregateOutput(f *testing.F) {
	mcABI := multicall3ABI(f)
	valid, err := mcABI.Methods["tryAggregate"].Outputs.Pack(testResults(newTestRand(5), 3))
	require.NoError(f, err)
	f.Add(valid)
	f.Fuzz(func(t *testing.T, out []byte) {
		decoded, err := decodeTryAggregateOutput(out)
		unpacked, abiErr := mcABI.Unpack("tryAggregate", out)
		if err != nil {
			return
		}
		require.NoError(t, abiErr)
		expected := *abi.ConvertType(unpacked[0], new([]multicall3.IMulticall3Result)).(*[]multicall3.IMulticall3Result)
		require.Equal(t, normalizeResults(expected), normalizeResults(decoded))
	})
}

// recordingNode answers eth_call with a fixed output and records the raw request bodies.
type recordingNode struct {
	mu     sync.Mutex
	bodies [][]byte
	output hexutil.Bytes
}

func (n *recordingNode) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var req struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
	}
	_ = json.Unmarshal(body, &req)
	n.mu.Lock()
	n.bodies = append(n.bodies, body)
	n.mu.Unlock()
	result := n.output
	if req.Method != "eth_call" {
		result = nil
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
}

func newRecordingNodeClient(t testing.TB, output []byte) (*recordingNode, *ethclient.EthClient) {
	node := &recordingNode{output: output}
	server := httptest.NewServer(node)
	t.Cleanup(server.Close)
	rpcClient, err := rpc.Dial(server.URL)
	require.NoError(t, err)
	t.Cleanup(rpcClient.Close)
	return node, ethclient.NewEthClient(rpcClient)
}

// The generated binding and the hand-encoded caller must put the same bytes on
// the wire and return the same values.
func TestMulticallCallerMatchesGeneratedBinding(t *testing.T) {
	mcABI := multicall3ABI(t)
	r := newTestRand(6)
	address := randomAddress(r)
	calls := testCalls(r, 50)
	results := testResults(r, 50)

	for _, opts := range []*bind.CallOpts{
		{Context: context.Background()},
		{Context: context.Background(), BlockNumber: big.NewInt(1234), From: randomAddress(r)},
	} {
		t.Run("tryBlockAndAggregate", func(t *testing.T) {
			out, err := mcABI.Methods["tryBlockAndAggregate"].Outputs.Pack(big.NewInt(99), common.Hash{9}, results)
			require.NoError(t, err)
			bindingNode, bindingClient := newRecordingNodeClient(t, out)
			binding, err := multicall3.NewMulticall3Caller(address, bindingClient)
			require.NoError(t, err)
			ourNode, ourClient := newRecordingNodeClient(t, out)

			bn1, bh1, res1, err1 := binding.ViewTryBlockAndAggregate(opts, false, calls)
			bn2, bh2, res2, err2 := newMulticallCaller(address, ourClient).ViewTryBlockAndAggregate(opts, false, calls)
			require.NoError(t, err1)
			require.NoError(t, err2)
			require.Len(t, ourNode.bodies, 1)
			require.Equal(t, string(bindingNode.bodies[0]), string(ourNode.bodies[0]))
			require.Zero(t, bn1.Cmp(bn2))
			require.Equal(t, bh1, bh2)
			require.Equal(t, normalizeResults(res1), normalizeResults(res2))
		})

		t.Run("tryAggregate", func(t *testing.T) {
			out, err := mcABI.Methods["tryAggregate"].Outputs.Pack(results)
			require.NoError(t, err)
			bindingNode, bindingClient := newRecordingNodeClient(t, out)
			binding, err := multicall3.NewMulticall3Caller(address, bindingClient)
			require.NoError(t, err)
			ourNode, ourClient := newRecordingNodeClient(t, out)

			res1, err1 := binding.ViewTryAggregate(opts, true, calls)
			res2, err2 := newMulticallCaller(address, ourClient).ViewTryAggregate(opts, true, calls)
			require.NoError(t, err1)
			require.NoError(t, err2)
			require.Len(t, ourNode.bodies, 1)
			require.Equal(t, string(bindingNode.bodies[0]), string(ourNode.bodies[0]))
			require.Equal(t, normalizeResults(res1), normalizeResults(res2))
		})
	}

	t.Run("no code", func(t *testing.T) {
		_, client := newRecordingNodeClient(t, nil)
		_, err := newMulticallCaller(address, client).ViewTryAggregate(&bind.CallOpts{}, false, calls)
		require.ErrorIs(t, err, bind.ErrNoCode)
	})
}

// staticBackend answers every call with the same output, for both callers.
type staticBackend struct {
	output []byte
}

func (b *staticBackend) CodeAt(context.Context, common.Address, *big.Int) ([]byte, error) {
	return []byte{1}, nil
}

func (b *staticBackend) CallContract(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
	return b.output, nil
}

func (b *staticBackend) CallContext(_ context.Context, result interface{}, _ string, _ ...interface{}) error {
	*result.(*hexutil.Bytes) = b.output
	return nil
}

func BenchmarkMulticallCallerEncodeDecode(b *testing.B) {
	mcABI := multicall3ABI(b)
	r := newTestRand(7)
	account := randomAddress(r)
	calls := make([]multicall3.IMulticall3Call, 2500)
	for i := range calls {
		calls[i] = multicall.BuildERC20BalanceCall(account, randomAddress(r))
	}
	out, err := mcABI.Methods["tryAggregate"].Outputs.Pack(testResults(r, len(calls)))
	require.NoError(b, err)
	opts := &bind.CallOpts{Context: context.Background()}

	b.Run("binding", func(b *testing.B) {
		binding, err := multicall3.NewMulticall3Caller(common.Address{1}, &staticBackend{output: out})
		require.NoError(b, err)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := binding.ViewTryAggregate(opts, false, calls)
			require.NoError(b, err)
		}
	})
	b.Run("handEncoded", func(b *testing.B) {
		caller := newMulticallCaller(common.Address{1}, &staticBackend{output: out})
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := caller.ViewTryAggregate(opts, false, calls)
			require.NoError(b, err)
		}
	})
}
