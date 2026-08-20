package activity

import (
	"reflect"
	"testing"

	eth "github.com/ethereum/go-ethereum/common"

	ac "github.com/status-im/status-go/pkg/services/wallet/activity/common"
)

// TODO #12120: cover missing cases
func TestFindUpdates(t *testing.T) {
	txIds := []ac.TransactionIdentity{
		ac.TransactionIdentity{
			ChainID: 1,
			Hash:    eth.HexToHash("0x1234"),
			Address: eth.HexToAddress("0x1234"),
		},
		ac.TransactionIdentity{
			ChainID: 1,
			Hash:    eth.HexToHash("0x5678"),
			Address: eth.HexToAddress("0x5678"),
		},
	}

	type findUpdatesResult struct {
		new     []mixedIdentityResult
		removed []EntryIdentity
	}

	tests := []struct {
		name       string
		identities []EntryIdentity
		updated    []Entry
		want       findUpdatesResult
	}{
		{
			name:       "Empty to single transaction update",
			identities: []EntryIdentity{},
			updated: []Entry{
				{payloadType: ac.SimpleTransactionPT, transaction: &txIds[0]},
			},
			want: findUpdatesResult{
				new: []mixedIdentityResult{{0, EntryIdentity{payloadType: ac.SimpleTransactionPT, transaction: &txIds[0]}}},
			},
		},
		{
			name: "No updates",
			identities: []EntryIdentity{
				EntryIdentity{
					payloadType: ac.SimpleTransactionPT, transaction: &txIds[0],
				},
			},
			updated: []Entry{
				{payloadType: ac.SimpleTransactionPT, transaction: &txIds[0]},
			},
			want: findUpdatesResult{},
		},
		{
			name:       "Empty to mixed updates",
			identities: []EntryIdentity{},
			updated: []Entry{
				{payloadType: ac.SimpleTransactionPT, transaction: &txIds[0]},
				{payloadType: ac.PendingTransactionPT, transaction: &txIds[1]},
			},
			want: findUpdatesResult{
				new: []mixedIdentityResult{{0, EntryIdentity{payloadType: ac.SimpleTransactionPT, transaction: &txIds[0]}},
					{1, EntryIdentity{payloadType: ac.PendingTransactionPT, transaction: &txIds[1]}},
				},
			},
		},
		{
			name: "Add one on top of one",
			identities: []EntryIdentity{
				EntryIdentity{
					payloadType: ac.SimpleTransactionPT, transaction: &txIds[0],
				},
			},
			updated: []Entry{
				{payloadType: ac.PendingTransactionPT, transaction: &txIds[1]},
				{payloadType: ac.SimpleTransactionPT, transaction: &txIds[0]},
			},
			want: findUpdatesResult{
				new: []mixedIdentityResult{{0, EntryIdentity{payloadType: ac.PendingTransactionPT, transaction: &txIds[1]}}},
			},
		},
		{
			name: "Add one on top keep window",
			identities: []EntryIdentity{
				EntryIdentity{payloadType: ac.SimpleTransactionPT, transaction: &txIds[0]},
				EntryIdentity{payloadType: ac.PendingTransactionPT, transaction: &txIds[1]},
			},
			updated: []Entry{
				{payloadType: ac.SimpleTransactionPT, transaction: &txIds[1]},
				{payloadType: ac.SimpleTransactionPT, transaction: &txIds[0]},
			},
			want: findUpdatesResult{
				new:     []mixedIdentityResult{{0, EntryIdentity{payloadType: ac.SimpleTransactionPT, transaction: &txIds[1]}}},
				removed: []EntryIdentity{EntryIdentity{payloadType: ac.PendingTransactionPT, transaction: &txIds[1]}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNew, gotRemoved := findUpdates(tt.identities, tt.updated)
			if !reflect.DeepEqual(gotNew, tt.want.new) || !reflect.DeepEqual(gotRemoved, tt.want.removed) {
				t.Errorf("findUpdates() = %v, %v, want %v, %v", gotNew, gotRemoved, tt.want.new, tt.want.removed)
			}
		})
	}
}
