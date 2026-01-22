package tokenhistoricalownership_test

import (
	"database/sql"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/status-im/status-go/internal/db/walletdatabase"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/services/accounts/accountsevent"
	af "github.com/status-im/status-go/services/wallet/activityfetcher"
	"github.com/status-im/status-go/services/wallet/tokenbalances"
	"github.com/status-im/status-go/services/wallet/tokenhistoricalownership"
	mock "github.com/status-im/status-go/services/wallet/tokenhistoricalownership/mock"
	"github.com/status-im/status-go/services/wallet/transferdetector"

	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/go-wallet-sdk/pkg/contracts/erc20"
	"github.com/status-im/go-wallet-sdk/pkg/contracts/erc721"
	"github.com/status-im/go-wallet-sdk/pkg/eventlog"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := testutils.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)
	return db, func() {
		require.NoError(t, db.Close())
	}
}

func TestServiceLifecycle(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	storage := tokenhistoricalownership.NewStorage(db)
	activityPublisher := pubsub.NewPublisher()
	balancesPublisher := pubsub.NewPublisher()
	transferPublisher := pubsub.NewPublisher()
	accountsPublisher := pubsub.NewPublisher()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockActivityProvider := mock.NewMockActivityTokenProvider(ctrl)
	mockBalancesProvider := mock.NewMockTokenBalancesProvider(ctrl)

	service := tokenhistoricalownership.NewService(
		db,
		storage,
		activityPublisher,
		mockActivityProvider,
		mockBalancesProvider,
		balancesPublisher,
		transferPublisher,
		accountsPublisher,
	)

	// Test Start
	err := service.Start()
	require.NoError(t, err)
	// Note: We can't access private fields (started, stopCh) from _test package
	// These assertions would need to be removed or tested indirectly

	// Test double Start (should be idempotent)
	err = service.Start()
	require.NoError(t, err)

	// Test Stop
	service.Stop()

	// Test double Stop (should be idempotent)
	service.Stop()
}

func TestGetPublisher(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	storage := tokenhistoricalownership.NewStorage(db)
	service := tokenhistoricalownership.NewService(
		db,
		storage,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	pub := service.GetPublisher()
	assert.NotNil(t, pub)
}

func TestMarkAsPreviouslyOwned(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	storage := tokenhistoricalownership.NewStorage(db)
	service := tokenhistoricalownership.NewService(
		db,
		storage,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	address := common.HexToAddress("0x1234567890123456789012345678901234567890")
	tokenAddress := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	chainID := uint64(1)

	// First call should return true (new entry)
	isNew, err := service.MarkAsPreviouslyOwned(address, chainID, tokenAddress)
	require.NoError(t, err)
	assert.True(t, isNew)

	// Second call should return false (already exists)
	isNew, err = service.MarkAsPreviouslyOwned(address, chainID, tokenAddress)
	require.NoError(t, err)
	assert.False(t, isNew)

	// Verify it's stored
	isOwned, err := service.IsOwned(address, chainID, tokenAddress)
	require.NoError(t, err)
	assert.True(t, isOwned)
}

func TestGetOwnedTokens(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	storage := tokenhistoricalownership.NewStorage(db)
	service := tokenhistoricalownership.NewService(
		db,
		storage,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	address := common.HexToAddress("0x1234567890123456789012345678901234567890")
	token1 := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	token2 := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	chainID1 := uint64(1)
	chainID2 := uint64(5)

	// Mark some tokens as owned
	_, err := service.MarkAsPreviouslyOwned(address, chainID1, token1)
	require.NoError(t, err)

	_, err = service.MarkAsPreviouslyOwned(address, chainID2, token2)
	require.NoError(t, err)

	// Get all owned tokens
	tokens, err := service.GetOwnedTokens(address)
	require.NoError(t, err)
	assert.Len(t, tokens, 2)

	// Verify token addresses are in the result
	addresses := []common.Address{tokens[0].TokenAddress, tokens[1].TokenAddress}
	assert.Contains(t, addresses, token1)
	assert.Contains(t, addresses, token2)
}

func TestGetOwnedTokensByChain(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	storage := tokenhistoricalownership.NewStorage(db)
	service := tokenhistoricalownership.NewService(
		db,
		storage,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	address := common.HexToAddress("0x1234567890123456789012345678901234567890")
	token1 := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	token2 := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	token3 := common.HexToAddress("0x6B175474E89094C44Da98b954EedeAC495271d0F")
	chainID1 := uint64(1)
	chainID2 := uint64(5)

	// Mark tokens as owned on different chains
	_, err := service.MarkAsPreviouslyOwned(address, chainID1, token1)
	require.NoError(t, err)

	_, err = service.MarkAsPreviouslyOwned(address, chainID1, token2)
	require.NoError(t, err)

	_, err = service.MarkAsPreviouslyOwned(address, chainID2, token3)
	require.NoError(t, err)

	// Get tokens for chain 1
	tokens, err := service.GetOwnedTokensByChain(address, chainID1)
	require.NoError(t, err)
	assert.Len(t, tokens, 2)

	// Verify token addresses
	addresses := []common.Address{tokens[0].TokenAddress, tokens[1].TokenAddress}
	assert.Contains(t, addresses, token1)
	assert.Contains(t, addresses, token2)

	// Get tokens for chain 2
	tokens, err = service.GetOwnedTokensByChain(address, chainID2)
	require.NoError(t, err)
	assert.Len(t, tokens, 1)
	assert.Equal(t, token3, tokens[0].TokenAddress)
}

func TestGetOwnedTokenKeys(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	storage := tokenhistoricalownership.NewStorage(db)
	service := tokenhistoricalownership.NewService(
		db,
		storage,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	address := common.HexToAddress("0x1234567890123456789012345678901234567890")
	token1 := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	chainID := uint64(1)

	// Mark token as owned
	_, err := service.MarkAsPreviouslyOwned(address, chainID, token1)
	require.NoError(t, err)

	// Get token keys
	keys, err := service.GetOwnedTokenKeys(address)
	require.NoError(t, err)
	assert.Len(t, keys, 1)
	assert.Equal(t, "1-0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", keys[0])
}

func TestHandleActivityDetected(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	storage := tokenhistoricalownership.NewStorage(db)
	activityPublisher := pubsub.NewPublisher()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockActivityProvider := mock.NewMockActivityTokenProvider(ctrl)

	service := tokenhistoricalownership.NewService(
		db,
		storage,
		activityPublisher,
		mockActivityProvider,
		nil,
		nil,
		nil,
		nil,
	)

	address := common.HexToAddress("0x1234567890123456789012345678901234567890")
	nativeToken := common.HexToAddress("0x0000000000000000000000000000000000000000")
	erc20Token := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	chainID := uint64(1)

	// Mock GetActivityTokens to return some tokens
	mockActivityProvider.EXPECT().
		GetActivityTokens(chainID, address).
		Return([]af.ActivityToken{
			{Address: nativeToken, IsNative: true},
			{Address: erc20Token, IsNative: false},
		}, nil)

	// Subscribe to ownership changed events
	pub := service.GetPublisher()
	ch, unsub := pubsub.Subscribe[tokenhistoricalownership.EventOwnershipChanged](pub, 10)
	defer unsub()

	// Start service to enable event processing
	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	// Give service time to start goroutines
	time.Sleep(10 * time.Millisecond)

	// Publish activity detected event (will be processed by handleActivityDetected)
	event := af.EventActivityDetected{
		ChainID: chainID,
		Account: address,
	}
	pubsub.Publish(activityPublisher, event)

	// Give it time to process
	time.Sleep(200 * time.Millisecond)

	// Verify tokens were marked as owned
	isOwned, err := service.IsOwned(address, chainID, nativeToken)
	require.NoError(t, err)
	assert.True(t, isOwned)

	isOwned, err = service.IsOwned(address, chainID, erc20Token)
	require.NoError(t, err)
	assert.True(t, isOwned)

	// Verify ownership changed event was published
	select {
	case publishedEvent := <-ch:
		assert.Equal(t, chainID, publishedEvent.ChainID)
		assert.Equal(t, address, publishedEvent.Account)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Event was not published")
	}
}

func TestHandleBalanceFetchFinished(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	storage := tokenhistoricalownership.NewStorage(db)
	balancesPublisher := pubsub.NewPublisher()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockBalancesProvider := mock.NewMockTokenBalancesProvider(ctrl)

	service := tokenhistoricalownership.NewService(
		db,
		storage,
		nil,
		nil,
		mockBalancesProvider,
		balancesPublisher,
		nil,
		nil,
	)

	address := common.HexToAddress("0x1234567890123456789012345678901234567890")
	nativeToken := common.HexToAddress("0x0000000000000000000000000000000000000000")
	erc20Token := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	chainID := uint64(1)

	// Mock GetAllBalancesForAccountAndChain to return some balances
	balances := map[tokenbalances.ContractAddress]*big.Int{
		tokenbalances.ContractAddress(nativeToken): big.NewInt(1000000000000000000), // 1 ETH
		tokenbalances.ContractAddress(erc20Token):  big.NewInt(1000000),             // 1 USDC
	}
	mockBalancesProvider.EXPECT().
		GetAllBalancesForAccountAndChain(gomock.Any(), tokenbalances.AccountAddress(address), chainID).
		Return(balances, tokenbalances.State{}, nil)

	// Subscribe to ownership changed events
	pub := service.GetPublisher()
	ch, unsub := pubsub.Subscribe[tokenhistoricalownership.EventOwnershipChanged](pub, 10)
	defer unsub()

	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	// Give service time to start goroutines
	time.Sleep(10 * time.Millisecond)

	// Publish balance fetch finished event
	event := tokenbalances.EventBalanceFetchFinished{
		ChainID:        chainID,
		Account:        address,
		BalanceChanged: true,
	}
	pubsub.Publish(balancesPublisher, event)

	// Give it time to process
	time.Sleep(200 * time.Millisecond)

	// Verify tokens with balance > 0 were marked as owned
	isOwned, err := service.IsOwned(address, chainID, nativeToken)
	require.NoError(t, err)
	assert.True(t, isOwned)

	isOwned, err = service.IsOwned(address, chainID, erc20Token)
	require.NoError(t, err)
	assert.True(t, isOwned)

	// Verify ownership changed event was published
	select {
	case publishedEvent := <-ch:
		assert.Equal(t, chainID, publishedEvent.ChainID)
		assert.Equal(t, address, publishedEvent.Account)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Event was not published")
	}
}

func TestHandleBalanceFetchFinished_NoChange(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	storage := tokenhistoricalownership.NewStorage(db)
	balancesPublisher := pubsub.NewPublisher()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockBalancesProvider := mock.NewMockTokenBalancesProvider(ctrl)

	service := tokenhistoricalownership.NewService(
		db,
		storage,
		nil,
		nil,
		mockBalancesProvider,
		balancesPublisher,
		nil,
		nil,
	)

	address := common.HexToAddress("0x1234567890123456789012345678901234567890")
	chainID := uint64(1)

	// Balance didn't change, so GetAllBalancesForAccountAndChain should NOT be called
	mockBalancesProvider.EXPECT().
		GetAllBalancesForAccountAndChain(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(0)

	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	// Publish balance fetch finished event with BalanceChanged = false
	event := tokenbalances.EventBalanceFetchFinished{
		ChainID:        chainID,
		Account:        address,
		BalanceChanged: false,
	}
	pubsub.Publish(balancesPublisher, event)

	// Give it time to process
	time.Sleep(50 * time.Millisecond)
}

func TestHandleTransferDetection_ERC20(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	storage := tokenhistoricalownership.NewStorage(db)
	transferPublisher := pubsub.NewPublisher()

	service := tokenhistoricalownership.NewService(
		db,
		storage,
		nil,
		nil,
		nil,
		nil,
		transferPublisher,
		nil,
	)

	recipient := common.HexToAddress("0x1234567890123456789012345678901234567890")
	tokenAddress := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	chainID := uint64(1)

	// Subscribe to ownership changed events
	pub := service.GetPublisher()
	ch, unsub := pubsub.Subscribe[tokenhistoricalownership.EventOwnershipChanged](pub, 10)
	defer unsub()

	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	// Give service time to start goroutines
	time.Sleep(10 * time.Millisecond)

	// Create ERC20 transfer event
	transferEvent := erc20.Erc20Transfer{
		From:  common.HexToAddress("0x0000000000000000000000000000000000000000"),
		To:    recipient,
		Value: big.NewInt(1000000),
		Raw: gethTypes.Log{
			Address: tokenAddress,
		},
	}

	// Publish transfer detection finished event
	event := transferdetector.EventTransferDetectionFinished{
		ChainID: chainID,
		Events: []eventlog.Event{
			{
				EventKey: eventlog.ERC20Transfer,
				Unpacked: transferEvent,
			},
		},
	}
	pubsub.Publish(transferPublisher, event)

	// Give it time to process
	time.Sleep(200 * time.Millisecond)

	// Verify token was marked as owned
	isOwned, err := service.IsOwned(recipient, chainID, tokenAddress)
	require.NoError(t, err)
	assert.True(t, isOwned)

	// Verify ownership changed event was published
	select {
	case publishedEvent := <-ch:
		assert.Equal(t, chainID, publishedEvent.ChainID)
		assert.Equal(t, recipient, publishedEvent.Account)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Event was not published")
	}
}

func TestHandleTransferDetection_ERC721(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	storage := tokenhistoricalownership.NewStorage(db)
	transferPublisher := pubsub.NewPublisher()

	service := tokenhistoricalownership.NewService(
		db,
		storage,
		nil,
		nil,
		nil,
		nil,
		transferPublisher,
		nil,
	)

	recipient := common.HexToAddress("0x1234567890123456789012345678901234567890")
	tokenAddress := common.HexToAddress("0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D")
	chainID := uint64(1)

	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	// Give service time to start goroutines
	time.Sleep(10 * time.Millisecond)

	// Create ERC721 transfer event
	transferEvent := erc721.Erc721Transfer{
		From:    common.HexToAddress("0x0000000000000000000000000000000000000000"),
		To:      recipient,
		TokenId: big.NewInt(1),
		Raw: gethTypes.Log{
			Address: tokenAddress,
		},
	}

	// Publish transfer detection finished event
	event := transferdetector.EventTransferDetectionFinished{
		ChainID: chainID,
		Events: []eventlog.Event{
			{
				EventKey: eventlog.ERC721Transfer,
				Unpacked: transferEvent,
			},
		},
	}
	pubsub.Publish(transferPublisher, event)

	// Give it time to process
	time.Sleep(200 * time.Millisecond)

	// Verify token was marked as owned
	isOwned, err := service.IsOwned(recipient, chainID, tokenAddress)
	require.NoError(t, err)
	assert.True(t, isOwned)
}

func TestHandleAccountsRemoved(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	storage := tokenhistoricalownership.NewStorage(db)
	accountsPublisher := pubsub.NewPublisher()

	service := tokenhistoricalownership.NewService(
		db,
		storage,
		nil,
		nil,
		nil,
		nil,
		nil,
		accountsPublisher,
	)

	address1 := common.HexToAddress("0x1234567890123456789012345678901234567890")
	address2 := common.HexToAddress("0x2234567890123456789012345678901234567890")
	tokenAddress := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	chainID := uint64(1)

	// Mark tokens as owned for both addresses
	_, err := service.MarkAsPreviouslyOwned(address1, chainID, tokenAddress)
	require.NoError(t, err)

	_, err = service.MarkAsPreviouslyOwned(address2, chainID, tokenAddress)
	require.NoError(t, err)

	// Verify both are owned
	isOwned, err := service.IsOwned(address1, chainID, tokenAddress)
	require.NoError(t, err)
	assert.True(t, isOwned)

	isOwned, err = service.IsOwned(address2, chainID, tokenAddress)
	require.NoError(t, err)
	assert.True(t, isOwned)

	err = service.Start()
	require.NoError(t, err)
	defer service.Stop()

	// Give service time to start goroutines
	time.Sleep(10 * time.Millisecond)

	// Publish account removed event
	event := accountsevent.AccountsRemovedEvent{
		Accounts: []common.Address{address1},
	}
	pubsub.Publish(accountsPublisher, event)

	// Give it time to process
	time.Sleep(200 * time.Millisecond)

	// Verify address1 tokens were removed
	isOwned, err = service.IsOwned(address1, chainID, tokenAddress)
	require.NoError(t, err)
	assert.False(t, isOwned)

	// Verify address2 tokens still exist
	isOwned, err = service.IsOwned(address2, chainID, tokenAddress)
	require.NoError(t, err)
	assert.True(t, isOwned)
}
