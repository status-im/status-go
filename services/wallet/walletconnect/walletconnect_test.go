package walletconnect

import (
	"reflect"
	"testing"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
)

func Test_sessionProposalToSupportedChain(t *testing.T) {
	type args struct {
		chains        []string
		supportsChain func(uint64) bool
	}
	tests := []struct {
		name          string
		args          args
		wantChains    []uint64
		wantEipChains []string
	}{
		{
			name: "filter_out_unsupported_chains_and_invalid_chains",
			args: args{
				chains: []string{"eip155:1", "eip155:3", "eip155:invalid"},
				supportsChain: func(chainID uint64) bool {
					return chainID == 1
				},
			},
			wantChains:    []uint64{1},
			wantEipChains: []string{"eip155:1"},
		},
		{
			name: "no_supported_chains",
			args: args{
				chains: []string{"eip155:3", "eip155:5"},
				supportsChain: func(chainID uint64) bool {
					return false
				},
			},
			wantChains:    []uint64{},
			wantEipChains: []string{},
		},
		{
			name: "empty_proposal",
			args: args{
				chains: []string{},
				supportsChain: func(chainID uint64) bool {
					return true
				},
			},
			wantChains:    []uint64{},
			wantEipChains: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotChains, gotEipChains := sessionProposalToSupportedChain(tt.args.chains, tt.args.supportsChain)
			if !reflect.DeepEqual(gotChains, tt.wantChains) {
				t.Errorf("sessionProposalToSupportedChain() gotChains = %v, want %v", gotChains, tt.wantChains)
			}
			if !reflect.DeepEqual(gotEipChains, tt.wantEipChains) {
				t.Errorf("sessionProposalToSupportedChain() gotEipChains = %v, want %v", gotEipChains, tt.wantEipChains)
			}
		})
	}
}

func Test_activeToOwnedAccounts(t *testing.T) {
	type args struct {
		activeAccounts []*accounts.Account
	}
	tests := []struct {
		name string
		args args
		want []types.Address
	}{
		{
			name: "filter_out_watch_accounts",
			args: args{
				activeAccounts: []*accounts.Account{
					{
						Address: types.HexToAddress("0x1"),
						Type:    accounts.AccountTypeWatch,
					},
					{
						Address: types.HexToAddress("0x2"),
						Type:    accounts.AccountTypeSeed,
					},
					{
						Address: types.HexToAddress("0x3"),
						Type:    accounts.AccountTypeSeed,
					},
				},
			},
			want: []types.Address{
				types.HexToAddress("0x2"),
				types.HexToAddress("0x3"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := activeToOwnedAccounts(tt.args.activeAccounts); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("activeToOwnedAccounts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_caip10Accounts(t *testing.T) {
	type args struct {
		addresses []types.Address
		chains    []uint64
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "generate_caip10_accounts",
			args: args{
				addresses: []types.Address{
					types.HexToAddress("0x1"),
					types.HexToAddress("0x2"),
				},
				chains: []uint64{1, 2},
			},
			want: []string{
				"eip155:1:0x0000000000000000000000000000000000000001",
				"eip155:2:0x0000000000000000000000000000000000000001",
				"eip155:1:0x0000000000000000000000000000000000000002",
				"eip155:2:0x0000000000000000000000000000000000000002",
			},
		},
		{
			name: "empty_addresses",
			args: args{
				addresses: []types.Address{},
				chains:    []uint64{1, 2},
			},
			want: []string{},
		},
		{
			name: "empty_chains",
			args: args{
				addresses: []types.Address{
					types.HexToAddress("0x1"),
					types.HexToAddress("0x2"),
				},
				chains: []uint64{},
			},
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := caip10Accounts(tt.args.addresses, tt.args.chains); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("caip10Accounts() = %v, want %v", got, tt.want)
			}
		})
	}
}
