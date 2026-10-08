package wallet

import (
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
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/status-im/go-wallet-sdk/pkg/balance/multistandardfetcher"
	"github.com/status-im/go-wallet-sdk/pkg/contracts/multicall3"
	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/rpc/chain/ethclient"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/pkg/services/wallet/multistandardbalance"
	mock_token "github.com/status-im/status-go/pkg/services/wallet/token/mock/token"
	tokenTypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
	"github.com/status-im/status-go/pkg/services/wallet/tokenbalances"
)

// These tests pin what the app gets for ERC20 balances: the balances the
// reader caches for the app (CacheBalances) and FetchOrGetCachedWalletBalances
// (Reader.GetCachedBalances), after real fetches through Fetcher ->
// go-wallet-sdk -> Controller -> storage against a scriptable Multicall3 node.
// They must give the same results on develop and after the balance fetch
// stopped storing zero balances.

const parityChainID = uint64(4663) // no mandatory tokens

var parityMulticall = common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")

func parityToken(i byte) common.Address { return common.Address{0x70, 19: i} }

// tokenAnswer is the node's answer to balanceOf on a token: a balance, or a failed call.
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

type parityAccounts struct{ accounts []cryptotypes.Address }

func (p parityAccounts) GetWalletAddresses() ([]cryptotypes.Address, error) { return p.accounts, nil }

type parityNetworks struct{}

func (parityNetworks) GetActiveNetworks() ([]*params.Network, error) {
	return []*params.Network{{ChainID: parityChainID}}, nil
}
func (parityNetworks) GetPublisher() *pubsub.Publisher { return pubsub.NewPublisher() }

// parityTokenList is the token list: what the balance fetch asks for and what
// the reader refreshes.
type parityTokenList struct {
	mu        sync.Mutex
	addresses []common.Address
}

func (p *parityTokenList) set(addresses ...common.Address) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.addresses = addresses
}

func (p *parityTokenList) GetTokenContractAddresses(uint64) ([]common.Address, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]common.Address(nil), p.addresses...), nil
}

func (p *parityTokenList) tokens() []*tokenTypes.Token {
	p.mu.Lock()
	defer p.mu.Unlock()
	tokens := make([]*tokenTypes.Token, len(p.addresses))
	for i, address := range p.addresses {
		tokens[i] = parityTokenOf(address)
	}
	return tokens
}

func parityTokenOf(address common.Address) *tokenTypes.Token {
	return &tokenTypes.Token{Token: &types.Token{ChainID: parityChainID, Address: address, Symbol: address.Hex()[38:], Decimals: 0}}
}

type parityCollectibles struct{}

func (parityCollectibles) GetCollectiblesList(uint64, common.Address) ([]multistandardbalance.CollectibleID, []multistandardbalance.CollectibleID, error) {
	return nil, nil, nil
}

type parityLastBlock struct{}

func (parityLastBlock) SetLatestBlockNumber(uint64, uint64) {}

// parityCache is the persisted token balances the app reads (token_balances
// table): upserted per account and token, entries with an error skipped.
type parityCache struct {
	mu       sync.Mutex
	balances map[common.Address]map[common.Address]tokenTypes.StorageToken
	saved    chan struct{}
}

func (c *parityCache) put(account, token common.Address, rawBalance string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.balances[account] == nil {
		c.balances[account] = make(map[common.Address]tokenTypes.StorageToken)
	}
	balance, _ := new(big.Float).SetString(rawBalance)
	c.balances[account][token] = tokenTypes.StorageToken{TokenAddress: token, TokenChainID: parityChainID, RawBalance: rawBalance, Balance: balance}
}

func (c *parityCache) get() (map[common.Address][]tokenTypes.StorageToken, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ret := make(map[common.Address][]tokenTypes.StorageToken)
	for account, tokens := range c.balances {
		for _, token := range tokens {
			ret[account] = append(ret[account], token)
		}
	}
	return ret, nil
}

func (c *parityCache) save(tokens map[common.Address][]tokenTypes.StorageToken) error {
	c.mu.Lock()
	for account, accountTokens := range tokens {
		for _, token := range accountTokens {
			if token.HasError {
				continue
			}
			if c.balances[account] == nil {
				c.balances[account] = make(map[common.Address]tokenTypes.StorageToken)
			}
			c.balances[account][token.TokenAddress] = token
		}
	}
	c.mu.Unlock()
	c.saved <- struct{}{}
	return nil
}

type parityHarness struct {
	t          *testing.T
	node       *parityNode
	tokenList  *parityTokenList
	cache      *parityCache
	controller *multistandardbalance.Controller
	reader     *Reader
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
		parityAccounts{accounts: []cryptotypes.Address{cryptotypes.BytesToAddress(account.Bytes())}}, nil, parityNetworks{},
		tokenList, parityCollectibles{}, parityLastBlock{}, nil, zap.NewNop())
	finished, unsubscribe := pubsub.Subscribe[multistandardbalance.EventBalanceFetchFinished](controller.GetPublisher(), 10)
	t.Cleanup(unsubscribe)

	cache := &parityCache{balances: make(map[common.Address]map[common.Address]tokenTypes.StorageToken), saved: make(chan struct{}, 10)}
	mockCtrl := gomock.NewController(t)
	tokenManager := mock_token.NewMockManagerInterface(mockCtrl)
	tokenManager.EXPECT().GetCachedBalances().DoAndReturn(cache.get).AnyTimes()
	tokenManager.EXPECT().CacheBalances(gomock.Any()).DoAndReturn(cache.save).AnyTimes()
	tokenManager.EXPECT().GetTokensByChains(gomock.Any()).DoAndReturn(func([]uint64) ([]*tokenTypes.Token, error) {
		return tokenList.tokens(), nil
	}).AnyTimes()
	tokenManager.EXPECT().GetTokensByKeys(gomock.Any()).DoAndReturn(func(keys []string) ([]*tokenTypes.Token, error) {
		tokens := make([]*tokenTypes.Token, 0, len(keys))
		for _, key := range keys {
			for i := byte(1); i < 16; i++ {
				if key == types.TokenKey(parityChainID, parityToken(i)) {
					tokens = append(tokens, parityTokenOf(parityToken(i)))
				}
			}
		}
		return tokens, nil
	}).AnyTimes()

	reader := NewReader(tokenManager, nil, &event.Feed{}, controller.GetPublisher(),
		tokenbalances.NewStorageMultistandardBalance(storage), pubsub.NewPublisher())
	require.NoError(t, reader.Start())
	t.Cleanup(reader.Stop)

	return &parityHarness{t: t, node: node, tokenList: tokenList, cache: cache, controller: controller,
		reader: reader, account: account, finished: finished}
}

// fetch runs one ERC20 fetch of the token list and reports whether the reader
// refreshed the app's cache (and so signalled the app) after it.
func (h *parityHarness) fetch(answers map[common.Address]tokenAnswer) bool {
	for drained := false; !drained; {
		select {
		case <-h.cache.saved:
		case <-time.After(300 * time.Millisecond):
			drained = true
		}
	}
	h.node.set(answers)
	key := multistandardbalance.BalancesKey{Account: h.account, ChainID: parityChainID}
	h.controller.TriggerFetchWithConfig(multistandardbalance.FetchConfig{key: {multistandardfetcher.ResultTypeERC20}})
	timeout := time.After(10 * time.Second)
	for done := false; !done; {
		select {
		case e := <-h.finished:
			done = e.Key == key && e.ResultType == multistandardfetcher.ResultTypeERC20
		case <-timeout:
			h.t.Fatal("ERC20 fetch did not finish")
		}
	}
	select {
	case <-h.cache.saved:
		return true
	case <-time.After(2 * time.Second):
		return false
	}
}

// appBalances is what FetchOrGetCachedWalletBalances returns for the account:
// raw balance per token, "error" for a token reported with an error.
func (h *parityHarness) appBalances() map[common.Address]string {
	tokens, err := h.reader.GetCachedBalances([]uint64{parityChainID}, []common.Address{h.account})
	require.NoError(h.t, err)
	ret := make(map[common.Address]string)
	for _, token := range tokens[h.account] {
		if token.HasError {
			ret[token.TokenAddress] = "error"
			continue
		}
		ret[token.TokenAddress] = token.RawBalance
	}
	return ret
}

func (h *parityHarness) cached() map[common.Address]string {
	h.cache.mu.Lock()
	defer h.cache.mu.Unlock()
	ret := make(map[common.Address]string)
	for token, balance := range h.cache.balances[h.account] {
		ret[token] = balance.RawBalance
	}
	return ret
}

func TestBalanceParity_ZeroBalanceReplacesStaleCache(t *testing.T) {
	h := newParityHarness(t)
	spent, neverHeld := parityToken(1), parityToken(2)
	h.cache.put(h.account, spent, "5") // persisted by an earlier session
	h.tokenList.set(spent, neverHeld)

	require.True(t, h.fetch(map[common.Address]tokenAnswer{spent: {balance: 0}, neverHeld: {balance: 0}}))
	require.Equal(t, map[common.Address]string{spent: "0"}, h.cached())
	require.Equal(t, map[common.Address]string{spent: "0"}, h.appBalances())
}

func TestBalanceParity_FailedCallKeepsLastKnownBalance(t *testing.T) {
	h := newParityHarness(t)
	held, cachedOnly, other := parityToken(1), parityToken(2), parityToken(3)
	h.cache.put(h.account, cachedOnly, "9")
	h.tokenList.set(held, cachedOnly, other)

	require.True(t, h.fetch(map[common.Address]tokenAnswer{held: {balance: 7}, cachedOnly: {fail: true}, other: {balance: 1}}))
	require.Equal(t, map[common.Address]string{held: "7", cachedOnly: "9", other: "1"}, h.appBalances())

	require.False(t, h.fetch(map[common.Address]tokenAnswer{held: {fail: true}, cachedOnly: {fail: true}, other: {balance: 1}}), "nothing changed")
	require.Equal(t, map[common.Address]string{held: "7", cachedOnly: "9", other: "1"}, h.appBalances())

	require.True(t, h.fetch(map[common.Address]tokenAnswer{held: {balance: 0}, cachedOnly: {fail: true}, other: {balance: 1}}))
	require.True(t, h.fetch(map[common.Address]tokenAnswer{held: {fail: true}, cachedOnly: {fail: true}, other: {balance: 2}}))
	require.Equal(t, map[common.Address]string{held: "0", cachedOnly: "9", other: "2"}, h.appBalances())
}

func TestBalanceParity_BalanceDropsToZero(t *testing.T) {
	h := newParityHarness(t)
	token := parityToken(1)
	h.tokenList.set(token)
	require.True(t, h.fetch(map[common.Address]tokenAnswer{token: {balance: 5}}))
	require.Equal(t, map[common.Address]string{token: "5"}, h.appBalances())
	require.True(t, h.fetch(map[common.Address]tokenAnswer{token: {balance: 0}}))
	require.Equal(t, map[common.Address]string{token: "0"}, h.cached())
	require.Equal(t, map[common.Address]string{token: "0"}, h.appBalances())
}

func TestBalanceParity_TokenListRebuiltBetweenFetchAndRead(t *testing.T) {
	h := newParityHarness(t)
	kept, dropped, cachedNeverAsked, added := parityToken(1), parityToken(2), parityToken(3), parityToken(4)
	h.cache.put(h.account, cachedNeverAsked, "4")
	h.cache.put(h.account, added, "6")
	h.tokenList.set(kept, dropped)
	require.True(t, h.fetch(map[common.Address]tokenAnswer{kept: {balance: 1}, dropped: {balance: 2}}))

	// The list is rebuilt: a token is dropped and one added, not fetched yet.
	h.tokenList.set(kept, added)
	expected := map[common.Address]string{kept: "1", dropped: "2", cachedNeverAsked: "4", added: "6"}
	require.Equal(t, expected, h.appBalances())

	// The next fetch asks the rebuilt list only.
	require.True(t, h.fetch(map[common.Address]tokenAnswer{kept: {balance: 1}, added: {balance: 0}}))
	expected[added] = "0"
	require.Equal(t, expected, h.appBalances())
	require.Equal(t, expected, h.cached())
}
