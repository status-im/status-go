package tokenbalances

import (
	"context"
	"maps"
	"math/big"
	"slices"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"go.uber.org/zap"

	"github.com/status-im/go-wallet-sdk/pkg/balance/multistandardfetcher"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/services/wallet/multistandardbalance"
	tokentypes "github.com/status-im/status-go/services/wallet/token/types"
	"github.com/status-im/status-go/signal"
)

// LoaderState represents the state of balance fetching for an account/chain
type LoaderState = int

const (
	LoaderStateIdle LoaderState = iota + 1
	LoaderStateUpdating
	LoaderStateError
)

type TokenBalancesChangedPayload struct {
	TokenBalances map[common.Address]map[common.Address]*hexutil.Big
}

// stateKey represents a unique key for account and chainID combination
type stateKey struct {
	Account common.Address
	ChainID uint64
}

type TokenListProvider interface {
	GetPublisher() *pubsub.Publisher
}

// Service handles token balance management, coordinating with multistandardBalanceController
// for balance fetching and emitting wallet events for balance changes.
type Service struct {
	storage                        Storage
	multistandardBalancePublisher  *pubsub.Publisher
	multistandardBalanceController *multistandardbalance.Controller
	publisher                      *pubsub.Publisher
	tokenListProvider              TokenListProvider

	loaderStates sync.Map // map[stateKey]LoaderState

	stopCh  chan struct{}
	started bool
	mu      sync.Mutex
}

// NewService creates a new tokenbalances service
func NewService(
	storage Storage,
	balancePublisher *pubsub.Publisher,
	balanceController *multistandardbalance.Controller,
	tokenListProvider TokenListProvider,
) *Service {
	return &Service{
		storage:                        storage,
		multistandardBalancePublisher:  balancePublisher,
		multistandardBalanceController: balanceController,
		tokenListProvider:              tokenListProvider,
		publisher:                      pubsub.NewPublisher(),
	}
}

// GetPublisher returns the publisher for token balance events
func (s *Service) GetPublisher() *pubsub.Publisher {
	return s.publisher
}

// Start begins watching for balance events
func (s *Service) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return nil
	}

	s.stopCh = make(chan struct{})
	s.started = true

	// Start watching balance events
	go s.watchBalanceEvents()

	// Start watching token list events
	go s.watchTokenListEvents()

	return nil
}

// Stop stops the service and cleans up resources
func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return
	}

	close(s.stopCh)
	s.stopCh = nil
	s.started = false
	s.loaderStates = sync.Map{}
}

// ForceRefresh triggers an immediate balance refresh for all wallet addresses
func (s *Service) ForceRefresh() error {
	s.multistandardBalanceController.TriggerFullFetch()
	return nil
}

// BalancesState represents the state and timestamp for a specific account/chain combination
type BalancesState struct {
	State         LoaderState
	AtBlockNumber *big.Int
	AtBlockHash   common.Hash
	FetchedAt     int64
}

// BalancesWithState contains balances and fetching state information
type BalancesWithState struct {
	Balances map[ContractAddress]*big.Int
	State    BalancesState
}

// BalancesWithStatePerChainID maps chainID to BalanceState
type BalancesWithStatePerChainID = map[uint64]BalancesWithState

// BalancesWithStatePerAddressAndChainID maps address to BalanceStatePerChainID
type BalancesWithStatePerAddressAndChainID = map[AccountAddress]BalancesWithStatePerChainID

// GetAllBalancesWithState returns the balances and state for each address and chainID
func (s *Service) GetAllBalancesWithState(ctx context.Context, chainIDs []uint64, addresses []common.Address) (BalancesWithStatePerAddressAndChainID, error) {
	result := make(BalancesWithStatePerAddressAndChainID)

	for _, address := range addresses {
		if _, ok := result[address]; !ok {
			result[address] = make(BalancesWithStatePerChainID)
		}

		for _, chainID := range chainIDs {
			storageBalances, storageState, err := s.storage.GetAllBalancesForAccountAndChain(ctx, address, chainID)
			if err != nil {
				return nil, err
			}
			// Get state for this address/chain combination
			key := stateKey{Account: address, ChainID: chainID}
			state := LoaderStateIdle // Default state
			if st, ok := s.loaderStates.Load(key); ok {
				if stInt, ok := st.(LoaderState); ok {
					state = stInt
				}
			}

			balancesWithState := BalancesWithState{
				Balances: storageBalances,
				State: BalancesState{
					State:         state,
					AtBlockNumber: storageState.AtBlockNumber,
					AtBlockHash:   storageState.AtBlockHash,
					FetchedAt:     storageState.FetchedAt,
				},
			}

			result[address][chainID] = balancesWithState
		}
	}

	return result, nil
}

// GetCachedBalances retrieves cached balances from storage without triggering a fetch
// Returns balances grouped by chainID, then by account address, then by token address
func (s *Service) GetCachedBalances(ctx context.Context, tokens []*tokentypes.Token, addresses []common.Address) (map[uint64]map[common.Address]map[common.Address]*hexutil.Big, error) {
	if s.storage == nil {
		return nil, nil
	}

	// Convert addresses to AccountAddress type
	accountAddresses := make([]AccountAddress, len(addresses))
	for i, addr := range addresses {
		accountAddresses[i] = AccountAddress(addr)
	}

	// Get balances from storage
	// Storage returns: map[chainID]map[account]map[token]*big.Int
	balances, err := s.storage.GetBalances(ctx, tokens, accountAddresses)
	if err != nil {
		return nil, err
	}

	// Convert to the expected return type
	result := make(map[uint64]map[common.Address]map[common.Address]*hexutil.Big)
	for chainID, accountBalances := range balances {
		result[chainID] = make(map[common.Address]map[common.Address]*hexutil.Big)
		for account, tokenBalances := range accountBalances {
			result[chainID][common.Address(account)] = make(map[common.Address]*hexutil.Big)
			for token, balance := range tokenBalances {
				result[chainID][common.Address(account)][common.Address(token)] = (*hexutil.Big)(balance)
			}
		}
	}

	return result, nil
}

// watchBalanceEvents subscribes to balance controller events and emits wallet signals
func (s *Service) watchBalanceEvents() {
	defer gocommon.LogOnPanic()

	if s.multistandardBalancePublisher == nil {
		logutils.ZapLogger().Warn("multistandardBalancePublisher is nil, balance events will not be monitored")
		return
	}

	// Subscribe to various balance events
	startedCh, unsubStarted := pubsub.Subscribe[multistandardbalance.EventBalanceFetchStarted](s.multistandardBalancePublisher, 10)
	finishedCh, unsubFinished := pubsub.Subscribe[multistandardbalance.EventBalanceFetchFinished](s.multistandardBalancePublisher, 10)
	errorCh, unsubError := pubsub.Subscribe[multistandardbalance.EventBalanceFetchError](s.multistandardBalancePublisher, 10)
	failedToStartCh, unsubFailedToStart := pubsub.Subscribe[multistandardbalance.EventBalanceFetchFailedToStart](s.multistandardBalancePublisher, 10)

	defer unsubStarted()
	defer unsubFinished()
	defer unsubError()
	defer unsubFailedToStart()

	for {
		select {
		case <-s.stopCh:
			return

		case event, ok := <-startedCh:
			if !ok {
				return
			}
			s.handleBalanceFetchStarted(event)

		case event, ok := <-finishedCh:
			if !ok {
				return
			}
			s.handleBalanceFetchFinished(event)

		case event, ok := <-errorCh:
			if !ok {
				return
			}
			s.handleBalanceFetchError(event)

		case event, ok := <-failedToStartCh:
			if !ok {
				return
			}
			s.handleBalanceFetchFailedToStart(event)
		}
	}
}

// handleBalanceFetchStarted processes balance fetch started events
func (s *Service) handleBalanceFetchStarted(event multistandardbalance.EventBalanceFetchStarted) {
	logutils.ZapLogger().Debug("balance fetch started",
		zap.Uint64("chainID", event.ChainID))

	addresses := make([]common.Address, 0)
	addresses = append(addresses, event.Config.Native...)
	addresses = slices.AppendSeq(addresses, maps.Keys(event.Config.ERC20))

	if len(addresses) == 0 {
		return
	}

	// Update state to LoaderStateUpdating for all addresses on this chain
	for _, address := range addresses {
		key := stateKey{Account: address, ChainID: event.ChainID}
		s.loaderStates.Store(key, LoaderStateUpdating)
	}

	// Emit wallet signal
	signal.SendWalletEvent(SignalFetchStarted, SignalFetchStartedPayload{
		ChainID:  event.ChainID,
		Accounts: addresses,
	})

	// Publish event
	if s.publisher != nil {
		pubsub.Publish(s.publisher, EventBalanceFetchStarted{
			ChainID:  event.ChainID,
			Accounts: addresses,
		})
	}
}

// handleBalanceFetchFailedToStart processes balance fetch failed to start events
func (s *Service) handleBalanceFetchFailedToStart(event multistandardbalance.EventBalanceFetchFailedToStart) {
	logutils.ZapLogger().Error("balance fetch failed to start",
		zap.Uint64("chainID", event.ChainID),
		zap.Error(event.Error))

	addresses := make([]common.Address, 0)
	addresses = append(addresses, event.Config.Native...)
	addresses = slices.AppendSeq(addresses, maps.Keys(event.Config.ERC20))

	if len(addresses) == 0 {
		return
	}

	// Update state to LoaderStateError for all addresses on this chain
	for _, address := range addresses {
		key := stateKey{Account: address, ChainID: event.ChainID}
		s.loaderStates.Store(key, LoaderStateError)
	}

	// Emit wallet signal
	signal.SendWalletEvent(SignalFetchFailedToStart, SignalFetchFailedToStartPayload{
		ChainID:  event.ChainID,
		Accounts: addresses,
		Error:    event.Error,
	})
}

// handleBalanceFetchFinished processes balance fetch finished events
func (s *Service) handleBalanceFetchFinished(event multistandardbalance.EventBalanceFetchFinished) {
	logutils.ZapLogger().Debug("balance fetch finished",
		zap.String("account", event.Key.Account.Hex()),
		zap.Uint64("chainID", event.Key.ChainID),
		zap.Bool("changed", event.BalanceChanged))

	if event.ResultType != multistandardfetcher.ResultTypeNative && event.ResultType != multistandardfetcher.ResultTypeERC20 {
		return
	}

	// Update state to LoaderStateIdle
	key := stateKey{Account: event.Key.Account, ChainID: event.Key.ChainID}
	s.loaderStates.Store(key, LoaderStateIdle)

	// Emit wallet signal
	signal.SendWalletEvent(SignalFetchFinished, SignalFetchFinishedPayload{
		ChainID:        event.Key.ChainID,
		Account:        event.Key.Account,
		BalanceChanged: event.BalanceChanged,
	})

	// Publish event
	if s.publisher != nil {
		pubsub.Publish(s.publisher, EventBalanceFetchFinished{
			ChainID:        event.Key.ChainID,
			Account:        event.Key.Account,
			BalanceChanged: event.BalanceChanged,
		})
	}
}

// handleBalanceFetchError processes balance fetch error events
func (s *Service) handleBalanceFetchError(event multistandardbalance.EventBalanceFetchError) {
	logutils.ZapLogger().Error("balance fetch error",
		zap.String("account", event.Key.Account.Hex()),
		zap.Uint64("chainID", event.Key.ChainID),
		zap.Error(event.Error))

	if event.ResultType != multistandardfetcher.ResultTypeNative && event.ResultType != multistandardfetcher.ResultTypeERC20 {
		return
	}

	// Update state to LoaderStateError
	key := stateKey{Account: event.Key.Account, ChainID: event.Key.ChainID}
	s.loaderStates.Store(key, LoaderStateError)

	// Emit wallet signal
	signal.SendWalletEvent(SignalFetchError, SignalFetchErrorPayload{
		ChainID: event.Key.ChainID,
		Account: event.Key.Account,
		Error:   event.Error,
	})

	// Publish event
	if s.publisher != nil {
		pubsub.Publish(s.publisher, EventBalanceFetchError{
			ChainID: event.Key.ChainID,
			Account: event.Key.Account,
			Error:   event.Error,
		})
	}
}

// watchTokenListEvents subscribes to token list events and emits wallet signals
func (s *Service) watchTokenListEvents() {
	defer gocommon.LogOnPanic()

	if s.tokenListProvider == nil {
		logutils.ZapLogger().Warn("tokenListProvider is nil, token list events will not be monitored")
		return
	}

	ch, unsub := pubsub.Subscribe[tokentypes.EventTokenListUpdated](s.tokenListProvider.GetPublisher(), 10)
	defer unsub()

	for {
		select {
		case <-s.stopCh:
			return
		case _, ok := <-ch:
			if !ok {
				return
			}
			// Force refresh balances to ensure we have balances for the new tokens
			_ = s.ForceRefresh()
		}
	}
}
