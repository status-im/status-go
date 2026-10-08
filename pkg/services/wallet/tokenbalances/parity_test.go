package tokenbalances_test

import (
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
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
	wsdktypes "github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/rpc/chain/ethclient"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/pkg/services/wallet/multistandardbalance"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
	"github.com/status-im/status-go/pkg/services/wallet/tokenbalances"
)

// These tests pin what the wallet reader gets from the live balance storage
// after real fetches (Fetcher -> go-wallet-sdk -> Controller -> storage ->
// GetBalances), against a scriptable Multicall3 node. They describe the
// behaviour before the balance fetch stopped storing zero balances and must
// hold unchanged after it.

const parityChainID = uint64(1)

var parityMulticall = common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")

func parityToken(i byte) common.Address { return common.Address{0x70, 19: i} }

// tokenAnswer is what the node answers for balanceOf on a token: a balance, or a failed call.
type tokenAnswer struct {
	balance int64
	fail    bool
}

type parityNode struct {
	mu      sync.Mutex
	answers map[common.Address]tokenAnswer
	block   int64
}

func (n *parityNode) set(answers map[common.Address]tokenAnswer) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.answers = answers
}

func (n *parityNode) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var req struct {
		ID     json.RawMessage   `json:"id"`
		Method string            `json:"method"`
		Params []json.RawMessage `json:"params"`
	}
	_ = json.Unmarshal(body, &req)
	reply := map[string]any{"jsonrpc": "2.0", "id": req.ID}
	out, err := n.call(req.Method, req.Params)
	if err != nil {
		reply["error"] = map[string]any{"code": -32000, "message": err.Error()}
	} else {
		reply["result"] = hexutil.Bytes(out)
	}
	_ = json.NewEncoder(w).Encode(reply)
}

func (n *parityNode) call(method string, rawParams []json.RawMessage) ([]byte, error) {
	if method == "eth_getCode" {
		return []byte{1}, nil
	}
	var msg struct {
		Data  hexutil.Bytes `json:"data"`
		Input hexutil.Bytes `json:"input"`
	}
	if err := json.Unmarshal(rawParams[0], &msg); err != nil {
		return nil, err
	}
	data := msg.Input
	if len(data) == 0 {
		data = msg.Data
	}
	mcABI, err := multicall3.Multicall3MetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	method3, err := mcABI.MethodById(data[:4])
	if err != nil {
		return nil, err
	}
	args, err := method3.Inputs.Unpack(data[4:])
	if err != nil {
		return nil, err
	}
	calls := *abi.ConvertType(args[1], new([]multicall3.IMulticall3Call)).(*[]multicall3.IMulticall3Call)

	n.mu.Lock()
	defer n.mu.Unlock()
	results := make([]multicall3.IMulticall3Result, len(calls))
	for i, call := range calls {
		answer, known := n.answers[call.Target]
		switch {
		case call.Target == parityMulticall: // getEthBalance
			results[i] = multicall3.IMulticall3Result{Success: true, ReturnData: make([]byte, 32)}
		case !known || answer.fail:
			results[i] = multicall3.IMulticall3Result{Success: false, ReturnData: []byte{}}
		default:
			results[i] = multicall3.IMulticall3Result{Success: true, ReturnData: common.LeftPadBytes(big.NewInt(answer.balance).Bytes(), 32)}
		}
	}
	if method3.Name == "tryBlockAndAggregate" {
		n.block++
		return method3.Outputs.Pack(big.NewInt(n.block), common.Hash{byte(n.block)}, results)
	}
	return method3.Outputs.Pack(results)
}

type parityEthClientGetter struct{ client ethclient.EthClientInterface }

func (g parityEthClientGetter) EthClient(uint64) (ethclient.EthClientInterface, error) {
	return g.client, nil
}

type parityAccounts struct{ accounts []types.Address }

func (p parityAccounts) GetWalletAddresses() ([]types.Address, error) { return p.accounts, nil }

type parityNetworks struct{}

func (parityNetworks) GetActiveNetworks() ([]*params.Network, error) {
	return []*params.Network{{ChainID: parityChainID}}, nil
}
func (parityNetworks) GetPublisher() *pubsub.Publisher { return pubsub.NewPublisher() }

type parityTokenList struct {
	mu     sync.Mutex
	tokens []common.Address
}

func (p *parityTokenList) set(tokens ...common.Address) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.tokens = tokens
}

func (p *parityTokenList) GetTokenContractAddresses(uint64) ([]common.Address, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]common.Address(nil), p.tokens...), nil
}

type parityCollectibles struct{}

func (parityCollectibles) GetCollectiblesList(uint64, common.Address) ([]multistandardbalance.CollectibleID, []multistandardbalance.CollectibleID, error) {
	return nil, nil, nil
}

type parityLastBlock struct{}

func (parityLastBlock) SetLatestBlockNumber(uint64, uint64) {}

type parityHarness struct {
	t          *testing.T
	node       *parityNode
	tokenList  *parityTokenList
	controller *multistandardbalance.Controller
	reader     *tokenbalances.StorageMultistandardBalance
	account    common.Address
	finished   <-chan multistandardbalance.EventBalanceFetchFinished
}

func newParityHarness(t *testing.T) *parityHarness {
	node := &parityNode{}
	server := httptest.NewServer(node)
	t.Cleanup(server.Close)
	rpcClient, err := rpc.Dial(server.URL)
	require.NoError(t, err)
	t.Cleanup(rpcClient.Close)

	account := common.Address{0xac, 19: 1}
	tokenList := &parityTokenList{}
	storage := multistandardbalance.NewStorageMemory()
	fetcher := multistandardbalance.NewFetcher(parityEthClientGetter{client: ethclient.NewEthClient(rpcClient)},
		multistandardbalance.DefaultBatchSize, map[uint64]common.Address{parityChainID: parityMulticall})
	config := multistandardbalance.DefaultControllerConfig()
	config.FetchDebounceTime = time.Millisecond
	controller := multistandardbalance.NewController(config, storage, fetcher,
		parityAccounts{accounts: []types.Address{types.BytesToAddress(account.Bytes())}}, nil, parityNetworks{},
		tokenList, parityCollectibles{}, parityLastBlock{}, nil, zap.NewNop())
	finished, unsubscribe := pubsub.Subscribe[multistandardbalance.EventBalanceFetchFinished](controller.GetPublisher(), 10)
	t.Cleanup(unsubscribe)

	return &parityHarness{t: t, node: node, tokenList: tokenList, controller: controller,
		reader: tokenbalances.NewStorageMultistandardBalance(storage), account: account, finished: finished}
}

// fetch runs one ERC20 fetch of the current token list against the given answers.
func (h *parityHarness) fetch(answers map[common.Address]tokenAnswer) {
	h.node.set(answers)
	key := multistandardbalance.BalancesKey{Account: h.account, ChainID: parityChainID}
	h.controller.TriggerFetchWithConfig(multistandardbalance.FetchConfig{key: {multistandardfetcher.ResultTypeERC20}})
	timeout := time.After(10 * time.Second)
	for {
		select {
		case event := <-h.finished:
			if event.Key == key && event.ResultType == multistandardfetcher.ResultTypeERC20 {
				return
			}
		case <-timeout:
			h.t.Fatal("ERC20 fetch did not finish")
		}
	}
}

// read returns what the reader gets for the tokens: a balance, nil (unknown), or absent.
func (h *parityHarness) read(tokens ...common.Address) map[common.Address]*big.Int {
	list := make([]*tokentypes.Token, len(tokens))
	for i, token := range tokens {
		list[i] = &tokentypes.Token{Token: &wsdktypes.Token{ChainID: parityChainID, Address: token}}
	}
	balances, err := h.reader.GetBalances(context.Background(), list, []common.Address{h.account})
	require.NoError(h.t, err)
	return balances[parityChainID][h.account]
}

func requireBalance(t *testing.T, balances map[common.Address]*big.Int, token common.Address, expected int64) {
	t.Helper()
	balance, present := balances[token]
	require.True(t, present, "%s present", token.Hex())
	require.NotNil(t, balance, "%s has a value", token.Hex())
	require.Equal(t, expected, balance.Int64(), token.Hex())
}

func requireUnknown(t *testing.T, balances map[common.Address]*big.Int, token common.Address) {
	t.Helper()
	balance, present := balances[token]
	require.True(t, present, "%s present", token.Hex())
	require.Nil(t, balance, "%s unknown", token.Hex())
}

func TestReaderParity_ZeroBalance(t *testing.T) {
	h := newParityHarness(t)
	zero := parityToken(1)
	h.tokenList.set(zero)
	h.fetch(map[common.Address]tokenAnswer{zero: {balance: 0}})
	requireBalance(t, h.read(zero), zero, 0)
}

func TestReaderParity_FailedCall(t *testing.T) {
	h := newParityHarness(t)
	held, neverAnswered := parityToken(1), parityToken(2)
	h.tokenList.set(held, neverAnswered)
	h.fetch(map[common.Address]tokenAnswer{held: {balance: 7}, neverAnswered: {fail: true}})
	balances := h.read(held, neverAnswered)
	requireBalance(t, balances, held, 7)
	requireUnknown(t, balances, neverAnswered)

	h.fetch(map[common.Address]tokenAnswer{held: {fail: true}, neverAnswered: {fail: true}})
	balances = h.read(held, neverAnswered)
	requireBalance(t, balances, held, 7)
	requireUnknown(t, balances, neverAnswered)

	zeroThenFailed := parityToken(3)
	h.tokenList.set(held, zeroThenFailed)
	h.fetch(map[common.Address]tokenAnswer{held: {balance: 7}, zeroThenFailed: {balance: 0}})
	h.fetch(map[common.Address]tokenAnswer{held: {balance: 7}, zeroThenFailed: {fail: true}})
	requireBalance(t, h.read(zeroThenFailed), zeroThenFailed, 0)
}

func TestReaderParity_NeverAskedToken(t *testing.T) {
	h := newParityHarness(t)
	asked, neverAsked := parityToken(1), parityToken(2)
	h.tokenList.set(asked)
	h.fetch(map[common.Address]tokenAnswer{asked: {balance: 3}})
	balances := h.read(asked, neverAsked)
	requireBalance(t, balances, asked, 3)
	requireUnknown(t, balances, neverAsked)
}

func TestReaderParity_BalanceDropsToZero(t *testing.T) {
	h := newParityHarness(t)
	token := parityToken(1)
	h.tokenList.set(token)
	h.fetch(map[common.Address]tokenAnswer{token: {balance: 5}})
	requireBalance(t, h.read(token), token, 5)
	h.fetch(map[common.Address]tokenAnswer{token: {balance: 0}})
	requireBalance(t, h.read(token), token, 0)
}

func TestReaderParity_TokenListRebuild(t *testing.T) {
	h := newParityHarness(t)
	kept, dropped, droppedZero, added := parityToken(1), parityToken(2), parityToken(3), parityToken(4)
	h.tokenList.set(kept, dropped, droppedZero)
	h.fetch(map[common.Address]tokenAnswer{kept: {balance: 1}, dropped: {balance: 2}, droppedZero: {balance: 0}})

	// The list is rebuilt between the fetch and the read.
	h.tokenList.set(kept, added)
	balances := h.read(kept, dropped, droppedZero, added)
	requireBalance(t, balances, kept, 1)
	requireBalance(t, balances, dropped, 2)
	requireBalance(t, balances, droppedZero, 0)
	requireUnknown(t, balances, added)

	// The next fetch asks the rebuilt list only.
	h.fetch(map[common.Address]tokenAnswer{kept: {balance: 1}, added: {balance: 0}})
	balances = h.read(kept, dropped, droppedZero, added)
	requireBalance(t, balances, kept, 1)
	requireBalance(t, balances, dropped, 2)
	requireBalance(t, balances, droppedZero, 0)
	requireBalance(t, balances, added, 0)
}
