package history

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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
