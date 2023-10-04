package history

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"
)

func newTestDB(t *testing.T) *BalanceDB {
	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)
	return NewBalanceDB(db)
}

func dbWithEntries(t *testing.T, entries []*entry) *BalanceDB {
	db := newTestDB(t)
	for _, entry := range entries {
		err := db.add(entry)
		require.NoError(t, err)
	}
	return db
}

func TestBalance_addPaddingPoints(t *testing.T) {
	type args struct {
		currency         string
		address          common.Address
		fromTimestamp    uint64
		currentTimestamp uint64
		data             []*entry
		limit            int
	}
	tests := []struct {
		name    string
		args    args
		want    []*entry
		wantErr bool
	}{
		{
			name: "addOnePaddingPointAtMiddle",
			args: args{
				currency:         "ETH",
				address:          common.Address{1},
				fromTimestamp:    0,
				currentTimestamp: 2,
				data: []*entry{
					{
						balance:     big.NewInt(0),
						timestamp:   0,
						tokenSymbol: "ETH",
						address:     common.Address{1},
					},
					{
						balance:     big.NewInt(2),
						timestamp:   2,
						tokenSymbol: "ETH",
						address:     common.Address{1},
					},
				},
				limit: 3,
			},
			want: []*entry{
				{
					balance:     big.NewInt(0),
					timestamp:   0,
					tokenSymbol: "ETH",
					address:     common.Address{1},
				},
				{
					balance:     big.NewInt(0),
					timestamp:   1,
					tokenSymbol: "ETH",
					address:     common.Address{1},
				},
				{
					balance:     big.NewInt(2),
					timestamp:   2,
					tokenSymbol: "ETH",
					address:     common.Address{1},
				},
			},
			wantErr: false,
		},
		{
			name: "noPaddingEqualsLimit",
			args: args{
				currency:         "ETH",
				address:          common.Address{1},
				fromTimestamp:    0,
				currentTimestamp: 2,
				data: []*entry{
					{
						balance:     big.NewInt(0),
						timestamp:   0,
						block:       big.NewInt(1),
						tokenSymbol: "ETH",
						address:     common.Address{1},
					},
					{
						balance:     big.NewInt(1),
						timestamp:   2,
						block:       big.NewInt(2),
						tokenSymbol: "ETH",
						address:     common.Address{1},
					},
				},
				limit: 2,
			},
			want: []*entry{
				{
					balance:     big.NewInt(0),
					timestamp:   0,
					block:       big.NewInt(1),
					tokenSymbol: "ETH",
					address:     common.Address{1},
				},
				{
					balance:     big.NewInt(1),
					timestamp:   2,
					block:       big.NewInt(2),
					tokenSymbol: "ETH",
					address:     common.Address{1},
				},
			},
			wantErr: false,
		},
		{
			name: "limitLessThanDataSize",
			args: args{
				currency:         "ETH",
				address:          common.Address{1},
				fromTimestamp:    0,
				currentTimestamp: 2,
				data: []*entry{
					{
						balance:     big.NewInt(0),
						timestamp:   0,
						block:       big.NewInt(1),
						tokenSymbol: "ETH",
						address:     common.Address{1},
					},
					{
						balance:     big.NewInt(1),
						timestamp:   2,
						block:       big.NewInt(2),
						tokenSymbol: "ETH",
						address:     common.Address{1},
					},
				},
				limit: 1,
			},
			want: []*entry{
				{
					balance:     big.NewInt(0),
					timestamp:   0,
					block:       big.NewInt(1),
					tokenSymbol: "ETH",
					address:     common.Address{1},
				},
				{
					balance:     big.NewInt(1),
					timestamp:   2,
					block:       big.NewInt(2),
					tokenSymbol: "ETH",
					address:     common.Address{1},
				},
			},
			wantErr: false,
		},
		{
			name: "addMultiplePaddingPoints",
			args: args{
				currency:         "ETH",
				address:          common.Address{1},
				fromTimestamp:    1,
				currentTimestamp: 5,
				data: []*entry{
					{
						balance:     big.NewInt(0),
						timestamp:   1,
						tokenSymbol: "ETH",
						address:     common.Address{1},
					},
					{
						balance:     big.NewInt(4),
						timestamp:   4,
						tokenSymbol: "ETH",
						address:     common.Address{1},
					},
					{
						balance:     big.NewInt(5),
						timestamp:   5,
						tokenSymbol: "ETH",
						address:     common.Address{1},
					},
				},
				limit: 5,
			},
			want: []*entry{
				{
					balance:     big.NewInt(0),
					timestamp:   1,
					tokenSymbol: "ETH",
					address:     common.Address{1},
				},
				{
					balance:     big.NewInt(0),
					timestamp:   2,
					tokenSymbol: "ETH",
					address:     common.Address{1},
				},
				{
					balance:     big.NewInt(0),
					timestamp:   3,
					tokenSymbol: "ETH",
					address:     common.Address{1},
				},
				{
					balance:     big.NewInt(4),
					timestamp:   4,
					tokenSymbol: "ETH",
					address:     common.Address{1},
				},
				{
					balance:     big.NewInt(5),
					timestamp:   5,
					tokenSymbol: "ETH",
					address:     common.Address{1},
				},
			},
			wantErr: false,
		},
		{
			name: "addMultiplePaddingPointsDuplicateTimestamps",
			args: args{
				currency:         "ETH",
				address:          common.Address{1},
				fromTimestamp:    1,
				currentTimestamp: 5,
				data: []*entry{
					{
						balance:     big.NewInt(0),
						timestamp:   1,
						tokenSymbol: "ETH",
						address:     common.Address{1},
					},
					{
						balance:     big.NewInt(0),
						timestamp:   1,
						tokenSymbol: "ETH",
						address:     common.Address{1},
					},
					{
						balance:     big.NewInt(4),
						timestamp:   4,
						tokenSymbol: "ETH",
						address:     common.Address{1},
					},
					{
						balance:     big.NewInt(5),
						timestamp:   5,
						tokenSymbol: "ETH",
						address:     common.Address{1},
					},
				},
				limit: 5,
			},
			want: []*entry{
				{
					balance:     big.NewInt(0),
					timestamp:   1,
					tokenSymbol: "ETH",
					address:     common.Address{1},
				},
				{
					balance:     big.NewInt(0),
					timestamp:   1,
					tokenSymbol: "ETH",
					address:     common.Address{1},
				},
				{
					balance:     big.NewInt(0),
					timestamp:   2,
					tokenSymbol: "ETH",
					address:     common.Address{1},
				},
				{
					balance:     big.NewInt(4),
					timestamp:   4,
					tokenSymbol: "ETH",
					address:     common.Address{1},
				},
				{
					balance:     big.NewInt(5),
					timestamp:   5,
					tokenSymbol: "ETH",
					address:     common.Address{1},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRes, err := addPaddingPoints(tt.args.currency, tt.args.address, tt.args.currentTimestamp, tt.args.data, tt.args.limit)
			if (err != nil) != tt.wantErr {
				t.Errorf("Balance.addPaddingPoints() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotRes, tt.want) {
				t.Errorf("Balance.addPaddingPoints() = %v, want %v", gotRes, tt.want)
			}
		})
	}
}

func TestBalance_addEdgePoints(t *testing.T) {

	walletDB := newTestDB(t)

	type fields struct {
		db *BalanceDB
	}
	type args struct {
		chainID       uint64
		currency      string
		address       common.Address
		fromTimestamp uint64
		toTimestamp   uint64
		data          []*entry
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantRes []*entry
		wantErr bool
	}{
		{
			name: "addToEmptyData",
			fields: fields{
				db: walletDB,
			},
			args: args{
				chainID:       111,
				currency:      "SNT",
				address:       common.Address{1},
				fromTimestamp: 1,
				toTimestamp:   2,
				data:          []*entry{},
			},
			wantRes: []*entry{
				{
					chainID:     111,
					balance:     big.NewInt(0),
					timestamp:   1,
					tokenSymbol: "SNT",
					address:     common.Address{1},
				},
				{
					chainID:     111,
					balance:     big.NewInt(0),
					timestamp:   2,
					tokenSymbol: "SNT",
					address:     common.Address{1},
				},
			},
			wantErr: false,
		},
		{
			name: "addToEmptyDataSinceGenesis",
			fields: fields{
				db: walletDB,
			},
			args: args{
				chainID:       111,
				currency:      "SNT",
				address:       common.Address{1},
				fromTimestamp: 0, // will set to genesisTimestamp
				toTimestamp:   genesisTimestamp + 1,
				data:          []*entry{},
			},
			wantRes: []*entry{
				{
					chainID:     111,
					balance:     big.NewInt(0),
					timestamp:   genesisTimestamp,
					tokenSymbol: "SNT",
					address:     common.Address{1},
				},
				{
					chainID:     111,
					balance:     big.NewInt(0),
					timestamp:   genesisTimestamp + 1,
					tokenSymbol: "SNT",
					address:     common.Address{1},
				},
			},
			wantErr: false,
		},
		{
			name: "addToNonEmptyDataFromPreviousEntry",
			fields: fields{
				db: dbWithEntries(t, []*entry{
					{
						chainID:     111,
						balance:     big.NewInt(1),
						timestamp:   1,
						block:       big.NewInt(1),
						tokenSymbol: "SNT",
						address:     common.Address{1},
					},
				}),
			},
			args: args{
				chainID:       111,
				currency:      "SNT",
				address:       common.Address{1},
				fromTimestamp: 2,
				toTimestamp:   4,
				data: []*entry{
					{
						chainID:     111,
						balance:     big.NewInt(3),
						timestamp:   3,
						block:       big.NewInt(3),
						tokenSymbol: "SNT",
						address:     common.Address{1},
					},
					{
						chainID:     111,
						balance:     big.NewInt(2),
						timestamp:   4,
						block:       big.NewInt(4),
						tokenSymbol: "SNT",
						address:     common.Address{1},
					},
				},
			},
			wantRes: []*entry{
				{
					chainID:     111,
					balance:     big.NewInt(1),
					timestamp:   2,
					tokenSymbol: "SNT",
					address:     common.Address{1},
				},
				{
					chainID:     111,
					balance:     big.NewInt(3),
					timestamp:   3,
					block:       big.NewInt(3),
					tokenSymbol: "SNT",
					address:     common.Address{1},
				},
				{
					chainID:     111,
					balance:     big.NewInt(2),
					timestamp:   4,
					block:       big.NewInt(4),
					tokenSymbol: "SNT",
					address:     common.Address{1},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Balance{
				db: tt.fields.db,
			}
			gotRes, err := b.addEdgePoints(tt.args.chainID, tt.args.currency, tt.args.address, tt.args.fromTimestamp, tt.args.toTimestamp, tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Balance.addEdgePoints() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotRes, tt.wantRes) {
				t.Errorf("Balance.addEdgePoints() = \n%v,\nwant \n%v", gotRes, tt.wantRes)
			}
		})
	}
}
