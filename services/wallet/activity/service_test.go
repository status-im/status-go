package activity

import (
	"database/sql"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	eth "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"

	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/services/wallet/transfer"
	"github.com/status-im/status-go/services/wallet/walletevent"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockCollectiblesManager implements the collectibles.ManagerInterface
type mockCollectiblesManager struct {
	mock.Mock
}

func (m *mockCollectiblesManager) FetchAssetsByCollectibleUniqueID(uniqueIDs []thirdparty.CollectibleUniqueID) ([]thirdparty.FullCollectibleData, error) {
	args := m.Called(uniqueIDs)
	res := args.Get(0)
	if res == nil {
		return nil, args.Error(1)
	}
	return res.([]thirdparty.FullCollectibleData), args.Error(1)
}

// mockTokenManager implements the token.ManagerInterface
type mockTokenManager struct {
	mock.Mock
}

func (m *mockTokenManager) LookupTokenIdentity(chainID uint64, address eth.Address, native bool) *token.Token {
	args := m.Called(chainID, address, native)
	res := args.Get(0)
	if res == nil {
		return nil
	}
	return res.(*token.Token)
}

func (m *mockTokenManager) LookupToken(chainID *uint64, tokenSymbol string) (tkn *token.Token, isNative bool) {
	args := m.Called(chainID, tokenSymbol)
	return args.Get(0).(*token.Token), args.Bool(1)
}

func setupTestService(tb testing.TB) (service *Service, eventFeed *event.Feed, tokenMock *mockTokenManager, collectiblesMock *mockCollectiblesManager, close func()) {
	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(tb, err)

	eventFeed = new(event.Feed)
	tokenMock = &mockTokenManager{}
	collectiblesMock = &mockCollectiblesManager{}
	service = NewService(db, tokenMock, collectiblesMock, eventFeed, nil)

	return service, eventFeed, tokenMock, collectiblesMock, func() {
		require.NoError(tb, db.Close())
	}
}

type arg struct {
	chainID         common.ChainID
	tokenAddressStr string
	tokenIDStr      string
	tokenID         *big.Int
	tokenAddress    *eth.Address
}

// insertStubTransfersWithCollectibles will insert nil if tokenIDStr is empty
func insertStubTransfersWithCollectibles(t *testing.T, db *sql.DB, args []arg) {
	trs, _, _ := transfer.GenerateTestTransfers(t, db, 0, len(args))
	for i := range args {
		trs[i].ChainID = args[i].chainID
		if args[i].tokenIDStr == "" {
			args[i].tokenID = nil
		} else {
			args[i].tokenID = new(big.Int)
			args[i].tokenID.SetString(args[i].tokenIDStr, 0)
		}
		args[i].tokenAddress = new(eth.Address)
		*args[i].tokenAddress = eth.HexToAddress(args[i].tokenAddressStr)
		transfer.InsertTestTransferWithOptions(t, db, trs[i].To, &trs[i], &transfer.TestTransferOptions{
			TokenAddress: *args[i].tokenAddress,
			TokenID:      args[i].tokenID,
		})
	}
}

func TestService_UpdateCollectibleInfo(t *testing.T) {
	s, e, tM, c, close := setupTestService(t)
	defer close()

	args := []arg{
		{5, "0xA2838FDA19EB6EED3F8B9EFF411D4CD7D2DE0313", "0x0D", nil, nil},
		{5, "0xA2838FDA19EB6EED3F8B9EFF411D4CD7D2DE0313", "0x762AD3E4934E687F8701F24C7274E5209213FD6208FF952ACEB325D028866949", nil, nil},
		{5, "0x3d6afaa395c31fcd391fe3d562e75fe9e8ec7e6a", "", nil, nil},
		{5, "0xA2838FDA19EB6EED3F8B9EFF411D4CD7D2DE0313", "0x0F", nil, nil},
	}
	insertStubTransfersWithCollectibles(t, s.db, args)

	ch := make(chan walletevent.Event)
	sub := e.Subscribe(ch)

	// Expect one call for the fungible token
	tM.On("LookupTokenIdentity", uint64(5), eth.HexToAddress("0x3d6afaa395c31fcd391fe3d562e75fe9e8ec7e6a"), false).Return(
		&token.Token{
			ChainID: 5,
			Address: eth.HexToAddress("0x3d6afaa395c31fcd391fe3d562e75fe9e8ec7e6a"),
			Symbol:  "STT",
		}, false,
	).Once()
	c.On("FetchAssetsByCollectibleUniqueID", []thirdparty.CollectibleUniqueID{
		{
			ContractID: thirdparty.ContractID{
				ChainID: args[3].chainID,
				Address: *args[3].tokenAddress},
			TokenID: &bigint.BigInt{Int: args[3].tokenID},
		}, {
			ContractID: thirdparty.ContractID{
				ChainID: args[1].chainID,
				Address: *args[1].tokenAddress},
			TokenID: &bigint.BigInt{Int: args[1].tokenID},
		},
	}).Return([]thirdparty.FullCollectibleData{
		{
			CollectibleData: thirdparty.CollectibleData{
				Name:     "Test 2",
				ImageURL: "test://url/2"},
			CollectionData: nil,
		}, {
			CollectibleData: thirdparty.CollectibleData{
				Name:     "Test 1",
				ImageURL: "test://url/1"},
			CollectionData: nil,
		},
	}, nil).Once()

	s.FilterActivityAsync(0, allAddressesFilter(), allNetworksFilter(), Filter{}, 0, 3)

	filterResponseCount := 0
	var updates []EntryData

	for i := 0; i < 2; i++ {
		select {
		case res := <-ch:
			switch res.Type {
			case EventActivityFilteringDone:
				var payload FilterResponse
				err := json.Unmarshal([]byte(res.Message), &payload)
				require.NoError(t, err)
				require.Equal(t, ErrorCodeSuccess, payload.ErrorCode)
				require.Equal(t, 3, len(payload.Activities))
				filterResponseCount++
			case EventActivityFilteringUpdate:
				err := json.Unmarshal([]byte(res.Message), &updates)
				require.NoError(t, err)
			}
		case <-time.NewTimer(1 * time.Second).C:
			require.Fail(t, "timeout while waiting for event")
		}
	}

	require.Equal(t, 1, filterResponseCount)
	require.Equal(t, 2, len(updates))
	require.Equal(t, "Test 2", *updates[0].NftName)
	require.Equal(t, "test://url/2", *updates[0].NftURL)
	require.Equal(t, "Test 1", *updates[1].NftName)
	require.Equal(t, "test://url/1", *updates[1].NftURL)

	sub.Unsubscribe()
}

func TestService_UpdateCollectibleInfo_Error(t *testing.T) {
	s, e, _, c, close := setupTestService(t)
	defer close()

	args := []arg{
		{5, "0xA2838FDA19EB6EED3F8B9EFF411D4CD7D2DE0313", "0x762AD3E4934E687F8701F24C7274E5209213FD6208FF952ACEB325D028866949", nil, nil},
		{5, "0xA2838FDA19EB6EED3F8B9EFF411D4CD7D2DE0313", "0x0D", nil, nil},
	}
	insertStubTransfersWithCollectibles(t, s.db, args)

	ch := make(chan walletevent.Event)
	sub := e.Subscribe(ch)

	c.On("FetchAssetsByCollectibleUniqueID", mock.Anything).Return(nil, thirdparty.ErrChainIDNotSupported).Once()

	s.FilterActivityAsync(0, allAddressesFilter(), allNetworksFilter(), Filter{}, 0, 5)

	filterResponseCount := 0
	updatesCount := 0

	select {
	case res := <-ch:
		switch res.Type {
		case EventActivityFilteringDone:
			var payload FilterResponse
			err := json.Unmarshal([]byte(res.Message), &payload)
			require.NoError(t, err)
			require.Equal(t, ErrorCodeSuccess, payload.ErrorCode)
			require.Equal(t, 2, len(payload.Activities))
			filterResponseCount++
		case EventActivityFilteringUpdate:
			updatesCount++
		}
	case <-time.NewTimer(100 * time.Millisecond).C:
	}

	select {
	case res := <-ch:
		switch res.Type {
		case EventActivityFilteringDone:
			filterResponseCount++
		case EventActivityFilteringUpdate:
			updatesCount++
		}
	case <-time.NewTimer(100 * time.Microsecond).C:
	}

	require.Equal(t, 1, filterResponseCount)
	require.Equal(t, 0, updatesCount)

	sub.Unsubscribe()
}
