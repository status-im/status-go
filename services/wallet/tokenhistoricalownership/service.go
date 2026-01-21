package tokenhistoricalownership

//go:generate go tool mockgen -package=mock_tokenhistoricalownership -source=service.go -destination=mock/service.go

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"sync"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/contracts/erc1155"
	"github.com/status-im/go-wallet-sdk/pkg/contracts/erc20"
	"github.com/status-im/go-wallet-sdk/pkg/contracts/erc721"
	"github.com/status-im/go-wallet-sdk/pkg/eventlog"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/services/accounts/accountsevent"
	af "github.com/status-im/status-go/services/wallet/activityfetcher"
	"github.com/status-im/status-go/services/wallet/tokenbalances"
	"github.com/status-im/status-go/services/wallet/transferdetector"
	"github.com/status-im/status-go/signal"
)

type TokenBalancesProvider interface {
	GetAllBalancesForAccountAndChain(ctx context.Context, accountAddress tokenbalances.AccountAddress, chainID uint64) (map[tokenbalances.ContractAddress]*big.Int, tokenbalances.State, error)
}

type ActivityTokenProvider interface {
	GetActivityTokens(chainID uint64, address common.Address) ([]af.ActivityToken, error)
}

// Service tracks historical token ownership for wallet accounts
type Service struct {
	db                        *sql.DB
	storage                   StorageInterface
	activityFetcherPublisher  *pubsub.Publisher
	activityTokenProvider     ActivityTokenProvider
	tokenBalancesProvider     TokenBalancesProvider
	tokenBalancesPublisher    *pubsub.Publisher
	transferDetectorPublisher *pubsub.Publisher
	accountsPublisher         *pubsub.Publisher
	publisher                 *pubsub.Publisher

	stopCh  chan struct{}
	started bool
	mu      sync.Mutex
}

// NewService creates a new historical ownership tracking service
func NewService(
	db *sql.DB,
	storage StorageInterface,
	activityFetcherPublisher *pubsub.Publisher,
	activityTokenProvider ActivityTokenProvider,
	tokenBalancesProvider TokenBalancesProvider,
	tokenBalancesPublisher *pubsub.Publisher,
	transferDetectorPublisher *pubsub.Publisher,
	accountsPublisher *pubsub.Publisher,
) *Service {
	return &Service{
		db:                        db,
		storage:                   storage,
		activityFetcherPublisher:  activityFetcherPublisher,
		activityTokenProvider:     activityTokenProvider,
		tokenBalancesProvider:     tokenBalancesProvider,
		tokenBalancesPublisher:    tokenBalancesPublisher,
		transferDetectorPublisher: transferDetectorPublisher,
		accountsPublisher:         accountsPublisher,
		publisher:                 pubsub.NewPublisher(),
	}
}

// GetPublisher returns the publisher for ownership change events
func (s *Service) GetPublisher() *pubsub.Publisher {
	return s.publisher
}

// Start begins tracking token ownership
func (s *Service) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return nil
	}

	s.stopCh = make(chan struct{})
	s.started = true

	// Start watching for activity fetcher events
	go s.watchActivityFetcherEvents()

	// Start watching for balance events
	go s.watchTokenBalancesEvents()

	// Start watching for transfer events
	go s.watchTransferEvents()

	// Start watching for account removal events
	go s.watchAccountEvents()

	return nil
}

// Stop stops the service
func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return
	}

	close(s.stopCh)
	s.stopCh = nil
	s.started = false
}

// MarkAsPreviouslyOwned manually marks a token as owned by an address
// Returns true if this is a new entry, false if already existed
func (s *Service) MarkAsPreviouslyOwned(ownerAddress common.Address, chainID uint64, tokenAddress common.Address) (bool, error) {
	isNew, err := s.storage.MarkAsOwned(ownerAddress, chainID, tokenAddress)
	if err != nil {
		logutils.ZapLogger().Error("failed to mark token as owned",
			zap.String("owner", ownerAddress.Hex()),
			zap.Uint64("chainID", chainID),
			zap.String("token", tokenAddress.Hex()),
			zap.Error(err))
		return false, err
	}

	if isNew {
		logutils.ZapLogger().Debug("marked token as historically owned",
			zap.String("owner", ownerAddress.Hex()),
			zap.Uint64("chainID", chainID),
			zap.String("token", tokenAddress.Hex()))

		// Emit event and signal
		s.emitOwnershipChanged(chainID, ownerAddress)
	}

	return isNew, nil
}

// GetOwnedTokens returns all tokens ever owned by an address
func (s *Service) GetOwnedTokens(ownerAddress common.Address) ([]TokenOwnership, error) {
	return s.storage.GetOwnedTokens(ownerAddress)
}

// GetOwnedTokensByChain returns all tokens ever owned by an address on a specific chain
func (s *Service) GetOwnedTokensByChain(ownerAddress common.Address, chainID uint64) ([]TokenOwnership, error) {
	return s.storage.GetOwnedTokensByChain(ownerAddress, chainID)
}

// IsOwned checks if a token was ever owned by an address
func (s *Service) IsOwned(ownerAddress common.Address, chainID uint64, tokenAddress common.Address) (bool, error) {
	return s.storage.IsOwned(ownerAddress, chainID, tokenAddress)
}

// GetOwnedTokenKeys returns token keys (formatted as "chainID-tokenAddress") for all tokens ever owned by an address
func (s *Service) GetOwnedTokenKeys(ownerAddress common.Address) ([]string, error) {
	ownedTokens, err := s.storage.GetOwnedTokens(ownerAddress)
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(ownedTokens))
	for _, token := range ownedTokens {
		// Format as "chainID-tokenAddress"
		key := fmt.Sprintf("%d-%s", token.ChainID, token.TokenAddress.Hex())
		keys = append(keys, key)
	}

	return keys, nil
}

// watchActivityFetcherEvents listens for activity detection events and fetches activity from storage
func (s *Service) watchActivityFetcherEvents() {
	defer gocommon.LogOnPanic()

	if s.activityFetcherPublisher == nil {
		logutils.ZapLogger().Warn("activityFetcherPublisher is nil, token ownership tracking from activities disabled")
		return
	}

	ch, unsub := pubsub.Subscribe[af.EventActivityDetected](s.activityFetcherPublisher, 10)
	defer unsub()

	for {
		select {
		case <-s.stopCh:
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			s.handleActivityDetected(event)
		}
	}
}

// handleActivityDetected processes activity detection events and marks tokens as owned
func (s *Service) handleActivityDetected(event af.EventActivityDetected) {
	if s.activityTokenProvider == nil {
		logutils.ZapLogger().Warn("activityTokenProvider is nil, cannot get activity tokens")
		return
	}

	// Get tokens from activity provider
	tokens, err := s.activityTokenProvider.GetActivityTokens(event.ChainID, event.Account)
	if err != nil {
		logutils.ZapLogger().Error("failed to get activity tokens",
			zap.Uint64("chainID", event.ChainID),
			zap.String("account", event.Account.Hex()),
			zap.Error(err))
		return
	}

	// Track if any new tokens were added
	hasNewTokens := false

	// Mark each token as owned
	for _, token := range tokens {
		isNew, err := s.storage.MarkAsOwned(event.Account, event.ChainID, token.Address)
		if err != nil {
			logutils.ZapLogger().Error("failed to mark token as owned from activity",
				zap.String("account", event.Account.Hex()),
				zap.Uint64("chainID", event.ChainID),
				zap.String("token", token.Address.Hex()),
				zap.Bool("isNative", token.IsNative),
				zap.Error(err))
			continue
		}
		if isNew {
			hasNewTokens = true
		}
	}

	// Emit event and signal if any new tokens were added
	if hasNewTokens {
		s.emitOwnershipChanged(event.ChainID, event.Account)
	}
}

// watchBalanceEvents listens for balance events and marks tokens with balance > 0 as owned
func (s *Service) watchTokenBalancesEvents() {
	defer gocommon.LogOnPanic()

	if s.tokenBalancesPublisher == nil {
		logutils.ZapLogger().Warn("tokenBalancesPublisher is nil, token ownership tracking from token balances disabled")
		return
	}

	ch, unsub := pubsub.Subscribe[tokenbalances.EventBalanceFetchFinished](s.tokenBalancesPublisher, 10)
	defer unsub()

	for {
		select {
		case <-s.stopCh:
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			s.handleBalanceFetchFinished(event)
		}
	}
}

// handleBalanceFetchFinished processes balance fetch finished events for Native and ERC20 tokens with balance > 0
func (s *Service) handleBalanceFetchFinished(event tokenbalances.EventBalanceFetchFinished) {
	// Only process if balance changed
	if !event.BalanceChanged {
		return
	}

	// Query storage to get all balances for this account on this chain
	ctx := context.Background()
	balances, _, err := s.tokenBalancesProvider.GetAllBalancesForAccountAndChain(ctx, event.Account, event.ChainID)
	if err != nil {
		logutils.ZapLogger().Error("failed to get balances from storage",
			zap.String("account", event.Account.Hex()),
			zap.Uint64("chainID", event.ChainID),
			zap.Error(err))
		return
	}

	// Track if any new tokens were added
	hasNewTokens := false

	// Mark tokens with balance > 0 as owned
	for tokenAddress, balance := range balances {
		if balance != nil && balance.Sign() > 0 {
			isNew, err := s.storage.MarkAsOwned(event.Account, event.ChainID, tokenAddress)
			if err != nil {
				logutils.ZapLogger().Error("failed to mark token as owned from balance event",
					zap.String("account", event.Account.Hex()),
					zap.Uint64("chainID", event.ChainID),
					zap.String("token", tokenAddress.Hex()),
					zap.Error(err))
				continue
			}
			if isNew {
				hasNewTokens = true
			}
		}
	}

	// Emit event and signal if any new tokens were added
	if hasNewTokens {
		s.emitOwnershipChanged(event.ChainID, event.Account)
	}
}

// watchTransferEvents listens for transfer detection events to track token ownership
func (s *Service) watchTransferEvents() {
	defer gocommon.LogOnPanic()

	if s.transferDetectorPublisher == nil {
		logutils.ZapLogger().Warn("transferDetectorPublisher is nil, token ownership tracking from transfers disabled")
		return
	}

	ch, unsub := pubsub.Subscribe[transferdetector.EventTransferDetectionFinished](s.transferDetectorPublisher, 10)
	defer unsub()

	for {
		select {
		case <-s.stopCh:
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			s.handleTransferDetection(msg)
		}
	}
}

// handleTransferDetection processes transfer events to track token ownership
func (s *Service) handleTransferDetection(msg transferdetector.EventTransferDetectionFinished) {
	// Track which accounts had new tokens added
	accountsWithNewTokens := make(map[common.Address]bool)

	for _, event := range msg.Events {
		// Extract recipient address and token address based on event type
		var recipient common.Address
		var tokenAddress common.Address

		switch event.EventKey {
		case eventlog.ERC20Transfer:
			unpackedEvent, ok := event.Unpacked.(erc20.Erc20Transfer)
			if !ok {
				logutils.ZapLogger().Error("failed to unpack ERC20Transfer event")
				continue
			}
			recipient = unpackedEvent.To
			tokenAddress = unpackedEvent.Raw.Address

		case eventlog.ERC721Transfer:
			unpackedEvent, ok := event.Unpacked.(erc721.Erc721Transfer)
			if !ok {
				logutils.ZapLogger().Error("failed to unpack ERC721Transfer event")
				continue
			}
			recipient = unpackedEvent.To
			tokenAddress = unpackedEvent.Raw.Address

		case eventlog.ERC1155TransferSingle:
			unpackedEvent, ok := event.Unpacked.(erc1155.Erc1155TransferSingle)
			if !ok {
				logutils.ZapLogger().Error("failed to unpack ERC1155TransferSingle event")
				continue
			}
			recipient = unpackedEvent.To
			tokenAddress = unpackedEvent.Raw.Address

		case eventlog.ERC1155TransferBatch:
			unpackedEvent, ok := event.Unpacked.(erc1155.Erc1155TransferBatch)
			if !ok {
				logutils.ZapLogger().Error("failed to unpack ERC1155TransferBatch event")
				continue
			}
			recipient = unpackedEvent.To
			tokenAddress = unpackedEvent.Raw.Address

		default:
			// Not a token transfer event, skip
			continue
		}

		// Record the token ownership for the recipient
		isNew, err := s.storage.MarkAsOwned(recipient, msg.ChainID, tokenAddress)
		if err != nil {
			logutils.ZapLogger().Error("failed to mark token as owned from transfer event",
				zap.String("recipient", recipient.Hex()),
				zap.Uint64("chainID", msg.ChainID),
				zap.String("token", tokenAddress.Hex()),
				zap.Error(err))
			continue
		}
		if isNew {
			accountsWithNewTokens[recipient] = true
		}
	}

	// Emit events and signals for accounts that had new tokens added
	for account := range accountsWithNewTokens {
		s.emitOwnershipChanged(msg.ChainID, account)
	}
}

// watchAccountEvents listens for account removal events to cleanup ownership records
func (s *Service) watchAccountEvents() {
	defer gocommon.LogOnPanic()

	if s.accountsPublisher == nil {
		logutils.ZapLogger().Warn("accountsPublisher is nil, account cleanup disabled")
		return
	}

	ch, unsub := pubsub.Subscribe[accountsevent.AccountsRemovedEvent](s.accountsPublisher, 10)
	defer unsub()

	for {
		select {
		case <-s.stopCh:
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			s.handleAccountsRemoved(event)
		}
	}
}

// handleAccountsRemoved cleans up ownership records for removed accounts
func (s *Service) handleAccountsRemoved(event accountsevent.AccountsRemovedEvent) {
	for _, address := range event.Accounts {
		err := s.storage.RemoveOwnerRecords(address)
		if err != nil {
			logutils.ZapLogger().Error("failed to remove ownership records for account",
				zap.String("address", address.Hex()),
				zap.Error(err))
		} else {
			logutils.ZapLogger().Debug("removed ownership records for account",
				zap.String("address", address.Hex()))
		}
	}
}

// emitOwnershipChanged emits event and sends signal when ownership changes for an account/chain
func (s *Service) emitOwnershipChanged(chainID uint64, account common.Address) {
	// Emit event to publisher
	if s.publisher != nil {
		pubsub.Publish(s.publisher, EventOwnershipChanged{
			ChainID: chainID,
			Account: account,
		})
	}

	// Send wallet signal
	signal.SendWalletEvent(SignalOwnershipChanged, SignalOwnershipChangedPayload{
		ChainID: chainID,
		Account: account,
	})
}
