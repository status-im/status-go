package transfer

import (
	"database/sql"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"
)

func setupBlockRangesTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)
	return db, func() {
		require.NoError(t, db.Close())
	}
}

func TestBlockRangeSequentialDAO_updateTokenRange(t *testing.T) {
	walletDb, stop := setupBlockRangesTestDB(t)
	defer stop()

	type fields struct {
		db *sql.DB
	}
	type args struct {
		chainID       uint64
		account       common.Address
		newBlockRange *BlockRange
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			"testTokenBlockRange",
			fields{db: walletDb},
			args{
				chainID: 1,
				account: common.Address{},
				newBlockRange: &BlockRange{
					LastKnown: big.NewInt(1),
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BlockRangeSequentialDAO{
				db: tt.fields.db,
			}

			err := b.upsertRange(tt.args.chainID, tt.args.account, newEthTokensBlockRanges())
			require.NoError(t, err)

			if err := b.updateTokenRange(tt.args.chainID, tt.args.account, tt.args.newBlockRange); (err != nil) != tt.wantErr {
				t.Errorf("BlockRangeSequentialDAO.updateTokenRange() error = %v, wantErr %v", err, tt.wantErr)
			}

			ethTokensBlockRanges, err := b.getBlockRange(tt.args.chainID, tt.args.account)
			require.NoError(t, err)
			require.NotNil(t, ethTokensBlockRanges.tokens)
			require.Equal(t, tt.args.newBlockRange.LastKnown, ethTokensBlockRanges.tokens.LastKnown)
		})
	}
}

func TestBlockRangeSequentialDAO_updateEthRange(t *testing.T) {
	walletDb, stop := setupBlockRangesTestDB(t)
	defer stop()

	type fields struct {
		db *sql.DB
	}
	type args struct {
		chainID       uint64
		account       common.Address
		newBlockRange *BlockRange
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			"testEthBlockRange",
			fields{db: walletDb},
			args{
				chainID: 1,
				account: common.Address{},
				newBlockRange: &BlockRange{
					Start:      big.NewInt(2),
					FirstKnown: big.NewInt(1),
					LastKnown:  big.NewInt(3),
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BlockRangeSequentialDAO{
				db: tt.fields.db,
			}

			// Initial insert
			dummyBlockRange := NewBlockRange()
			dummyBlockRange.FirstKnown = big.NewInt(2) // To confirm that it is updated it must be greater than newBlockRange.FirstKnown
			if err := b.upsertEthRange(tt.args.chainID, tt.args.account, dummyBlockRange); (err != nil) != tt.wantErr {
				t.Errorf("BlockRangeSequentialDAO.upsertEthRange() insert error = %v, wantErr %v", err, tt.wantErr)
			}

			ethTokensBlockRanges, err := b.getBlockRange(tt.args.chainID, tt.args.account)
			require.NoError(t, err)
			require.NotNil(t, ethTokensBlockRanges.eth)
			require.Equal(t, dummyBlockRange.Start, ethTokensBlockRanges.eth.Start)
			require.Equal(t, dummyBlockRange.FirstKnown, ethTokensBlockRanges.eth.FirstKnown)
			require.Equal(t, dummyBlockRange.LastKnown, ethTokensBlockRanges.eth.LastKnown)

			// Update
			if err := b.upsertEthRange(tt.args.chainID, tt.args.account, tt.args.newBlockRange); (err != nil) != tt.wantErr {
				t.Errorf("BlockRangeSequentialDAO.upsertEthRange() update error = %v, wantErr %v", err, tt.wantErr)
			}

			ethTokensBlockRanges, err = b.getBlockRange(tt.args.chainID, tt.args.account)
			require.NoError(t, err)
			require.NotNil(t, ethTokensBlockRanges.eth)
			require.Equal(t, tt.args.newBlockRange.Start, ethTokensBlockRanges.eth.Start)
			require.Equal(t, tt.args.newBlockRange.LastKnown, ethTokensBlockRanges.eth.LastKnown)
			require.Equal(t, tt.args.newBlockRange.FirstKnown, ethTokensBlockRanges.eth.FirstKnown)
		})
	}
}
