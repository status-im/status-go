package history

import (
	"errors"
	"math/big"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/event"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/accounts/accountsevent"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/t/utils"
	"github.com/status-im/status-go/transactions/fake"
	"github.com/status-im/status-go/walletdatabase"
)

func Test_entriesToDataPoints(t *testing.T) {
	type args struct {
		data []*entry
	}
	tests := []struct {
		name    string
		args    args
		want    []*DataPoint
		wantErr bool
	}{
		{
			name: "zeroAllChainsSameTimestamp",
			args: args{
				data: []*entry{
					{
						chainID:   1,
						balance:   big.NewInt(0),
						timestamp: 1,
						block:     big.NewInt(1),
					},
					{
						chainID:   2,
						balance:   big.NewInt(0),
						timestamp: 1,
						block:     big.NewInt(5),
					},
				},
			},
			want: []*DataPoint{
				{
					Balance:   (*hexutil.Big)(big.NewInt(0)),
					Timestamp: 1,
				},
			},
			wantErr: false,
		},
		{
			name: "oneZeroAllChainsDifferentTimestamp",
			args: args{
				data: []*entry{
					{
						chainID:   2,
						balance:   big.NewInt(0),
						timestamp: 1,
						block:     big.NewInt(1),
					},
					{
						chainID:   1,
						balance:   big.NewInt(2),
						timestamp: 2,
						block:     big.NewInt(2),
					},
				},
			},
			want: []*DataPoint{
				{
					Balance:   (*hexutil.Big)(big.NewInt(0)),
					Timestamp: 1,
				},
				{
					Balance:   (*hexutil.Big)(big.NewInt(2)),
					Timestamp: 2,
				},
			},
			wantErr: false,
		},
		{
			name: "nonZeroAllChainsDifferentTimestamp",
			args: args{
				data: []*entry{
					{
						chainID:   2,
						balance:   big.NewInt(1),
						timestamp: 1,
					},
					{
						chainID:   1,
						balance:   big.NewInt(2),
						timestamp: 2,
					},
				},
			},
			want: []*DataPoint{
				{
					Balance:   (*hexutil.Big)(big.NewInt(1)),
					Timestamp: 1,
				},
				{
					Balance:   (*hexutil.Big)(big.NewInt(3)),
					Timestamp: 2,
				},
			},
			wantErr: false,
		},
		{
			name: "sameChainDifferentTimestamp",
			args: args{
				data: []*entry{
					{
						chainID:   1,
						balance:   big.NewInt(1),
						timestamp: 1,
						block:     big.NewInt(1),
					},
					{
						chainID:   1,
						balance:   big.NewInt(2),
						timestamp: 2,
						block:     big.NewInt(2),
					},
					{
						chainID:   1,
						balance:   big.NewInt(0),
						timestamp: 3,
					},
				},
			},
			want: []*DataPoint{
				{
					Balance:   (*hexutil.Big)(big.NewInt(1)),
					Timestamp: 1,
				},
				{
					Balance:   (*hexutil.Big)(big.NewInt(2)),
					Timestamp: 2,
				},
				{
					Balance:   (*hexutil.Big)(big.NewInt(0)),
					Timestamp: 3,
				},
			},
			wantErr: false,
		},
		{
			name: "sameChainDifferentTimestampOtherChainsEmpty",
			args: args{
				data: []*entry{
					{
						chainID:   1,
						balance:   big.NewInt(1),
						timestamp: 1,
						block:     big.NewInt(1),
					},
					{
						chainID:   1,
						balance:   big.NewInt(2),
						timestamp: 2,
						block:     big.NewInt(2),
					},
					{
						chainID:   2,
						balance:   big.NewInt(0),
						timestamp: 2,
						block:     big.NewInt(2),
					},
					{
						chainID:   1,
						balance:   big.NewInt(2),
						timestamp: 3,
					},
				},
			},
			want: []*DataPoint{
				{
					Balance:   (*hexutil.Big)(big.NewInt(1)),
					Timestamp: 1,
				},
				{
					Balance:   (*hexutil.Big)(big.NewInt(2)),
					Timestamp: 2,
				},
				{
					Balance:   (*hexutil.Big)(big.NewInt(2)),
					Timestamp: 3,
				},
			},
			wantErr: false,
		},
		{
			name: "onlyEdgePointsOnManyChainsWithPadding",
			args: args{
				data: []*entry{
					// Left edge - same timestamp
					{
						chainID:   1,
						balance:   big.NewInt(1),
						timestamp: 1,
					},
					{
						chainID:   2,
						balance:   big.NewInt(2),
						timestamp: 1,
					},
					{
						chainID:   3,
						balance:   big.NewInt(3),
						timestamp: 1,
					},
					// Padding
					{
						chainID:   0,
						balance:   big.NewInt(6),
						timestamp: 2,
					},
					{
						chainID:   0,
						balance:   big.NewInt(6),
						timestamp: 3,
					},
					{
						chainID:   0,
						balance:   big.NewInt(6),
						timestamp: 4,
					},
					// Right edge - same timestamp
					{
						chainID:   1,
						balance:   big.NewInt(1),
						timestamp: 5,
					},
					{
						chainID:   2,
						balance:   big.NewInt(2),
						timestamp: 5,
					},
					{
						chainID:   3,
						balance:   big.NewInt(3),
						timestamp: 5,
					},
				},
			},
			want: []*DataPoint{
				{
					Balance:   (*hexutil.Big)(big.NewInt(6)),
					Timestamp: 1,
				},
				{
					Balance:   (*hexutil.Big)(big.NewInt(6)),
					Timestamp: 2,
				},
				{
					Balance:   (*hexutil.Big)(big.NewInt(6)),
					Timestamp: 3,
				},
				{
					Balance:   (*hexutil.Big)(big.NewInt(6)),
					Timestamp: 4,
				},
				{
					Balance:   (*hexutil.Big)(big.NewInt(6)),
					Timestamp: 5,
				},
			},
			wantErr: false,
		},
		{
			name: "multipleAddresses",
			args: args{
				data: []*entry{
					{
						chainID:   2,
						balance:   big.NewInt(5),
						timestamp: 1,
						address:   common.Address{1},
					},
					{
						chainID:   1,
						balance:   big.NewInt(6),
						timestamp: 1,
						address:   common.Address{2},
					},
					{
						chainID:   1,
						balance:   big.NewInt(1),
						timestamp: 2,
						address:   common.Address{1},
					},
					{
						chainID:   1,
						balance:   big.NewInt(2),
						timestamp: 3,
						address:   common.Address{2},
					},
					{
						chainID:   1,
						balance:   big.NewInt(4),
						timestamp: 4,
						address:   common.Address{2},
					},
				},
			},
			want: []*DataPoint{
				{
					Balance:   (*hexutil.Big)(big.NewInt(11)),
					Timestamp: 1,
				},
				{
					Balance:   (*hexutil.Big)(big.NewInt(12)),
					Timestamp: 2,
				},
				{
					Balance:   (*hexutil.Big)(big.NewInt(8)),
					Timestamp: 3,
				},
				{
					Balance:   (*hexutil.Big)(big.NewInt(10)),
					Timestamp: 4,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := entriesToDataPoints(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("entriesToDataPoints() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("entriesToDataPoints() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_removeBalanceHistoryOnEventAccountRemoved(t *testing.T) {
	appDB, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)

	walletDB, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	accountsDB, err := accounts.NewDB(appDB)
	require.NoError(t, err)

	address := common.HexToAddress("0x1234")
	accountFeed := event.Feed{}
	walletFeed := event.Feed{}
	chainID := uint64(1)
	txServiceMockCtrl := gomock.NewController(t)
	server, _ := fake.NewTestServer(txServiceMockCtrl)
	client := gethrpc.DialInProc(server)
	rpcClient, _ := rpc.NewClient(client, chainID, params.UpstreamRPCConfig{}, nil, nil)
	rpcClient.UpstreamChainID = chainID

	service := NewService(walletDB, accountsDB, &accountFeed, &walletFeed, rpcClient, nil, nil, nil)

	// Insert balances for address
	database := service.balance.db
	err = database.add(&entry{
		chainID:     chainID,
		address:     address,
		block:       big.NewInt(1),
		balance:     big.NewInt(1),
		timestamp:   1,
		tokenSymbol: "ETH",
	})
	require.NoError(t, err)
	err = database.add(&entry{
		chainID:     chainID,
		address:     address,
		block:       big.NewInt(2),
		balance:     big.NewInt(2),
		tokenSymbol: "ETH",
		timestamp:   2,
	})
	require.NoError(t, err)

	entries, err := database.getNewerThan(&assetIdentity{chainID, []common.Address{address}, "ETH"}, 0)
	require.NoError(t, err)
	require.Len(t, entries, 2)

	// Start service
	service.startAccountWatcher()

	// Watching accounts must start before sending event.
	// To avoid running goroutine immediately and let the controller subscribe first,
	// use any delay.
	group := sync.WaitGroup{}
	group.Add(1)
	go func() {
		defer group.Done()
		time.Sleep(1 * time.Millisecond)

		accountFeed.Send(accountsevent.Event{
			Type:     accountsevent.EventTypeRemoved,
			Accounts: []common.Address{address},
		})

		err := utils.Eventually(func() error {
			entries, err := database.getNewerThan(&assetIdentity{1, []common.Address{address}, "ETH"}, 0)
			if err == nil && len(entries) == 0 {
				return nil
			}
			return errors.New("data is not removed")
		}, 100*time.Millisecond, 10*time.Millisecond)
		require.NoError(t, err)
	}()

	group.Wait()

	// Stop service
	txServiceMockCtrl.Finish()
	server.Stop()
	service.stopAccountWatcher()
}
