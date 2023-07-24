package activity

import (
	"testing"

	"github.com/status-im/status-go/services/wallet/async"

	"github.com/stretchr/testify/require"
)

func Test_makeTaskType(t *testing.T) {
	type args struct {
		firstRequestID   int32
		secondRequestID  int32
		firstOriginalID  int32
		secondOriginalID int32
		policy           async.ReplacementPolicy
	}
	tests := []struct {
		name             string
		args             args
		wantDifferentIDs bool
	}{
		{
			name: "Different requestID",
			args: args{
				firstRequestID:   1,
				secondRequestID:  2,
				firstOriginalID:  1,
				secondOriginalID: 1,
				policy:           async.ReplacementPolicyCancelOld,
			},
			wantDifferentIDs: true,
		},
		{
			name: "Different originalID",
			args: args{
				firstRequestID:   1,
				secondRequestID:  1,
				firstOriginalID:  2,
				secondOriginalID: 3,
				policy:           async.ReplacementPolicyCancelOld,
			},
			wantDifferentIDs: true,
		},
		{
			name: "Same requestID and originalID",
			args: args{
				firstRequestID:   1,
				secondRequestID:  1,
				firstOriginalID:  1,
				secondOriginalID: 1,
				policy:           async.ReplacementPolicyCancelOld,
			},
			wantDifferentIDs: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			firstTT := makeTaskType(tt.args.firstRequestID, tt.args.firstOriginalID, tt.args.policy)
			secondTT := makeTaskType(tt.args.secondRequestID, tt.args.secondOriginalID, tt.args.policy)
			if tt.wantDifferentIDs {
				require.NotEqual(t, firstTT.ID, secondTT.ID)
			} else {
				require.Equal(t, firstTT.ID, secondTT.ID)
			}
			require.Equal(t, firstTT.Policy, secondTT.Policy)
		})
	}
}
