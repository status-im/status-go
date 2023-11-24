package walletconnect

import (
	"strings"
	"testing"
)

func Test_parseCaip2(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name      string
		args      args
		wantChain uint64
		wantErr   bool
	}{
		{
			name: "valid",
			args: args{
				str: "eip155:5",
			},
			wantChain: 5,
			wantErr:   false,
		},
		{
			name: "invalid_number",
			args: args{
				str: "eip155:5a",
			},
			wantChain: 0,
			wantErr:   true,
		},
		{
			name: "invalid_caip2_too_many",
			args: args{
				str: "eip155:1:5",
			},
			wantChain: 0,
			wantErr:   true,
		},
		{
			name: "invalid_caip2_not_enough",
			args: args{
				str: "eip1551",
			},
			wantChain: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNamespaceName, gotChainID, err := parseCaip2ChainID(tt.args.str)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCaip2ChainID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !strings.Contains(tt.args.str, gotNamespaceName) {
				t.Errorf("parseCaip2ChainID() = %v, doesn't match %v", gotNamespaceName, tt.args.str)
			}
			if gotChainID != tt.wantChain {
				t.Errorf("parseCaip2ChainID() = %v, want %v", gotChainID, tt.wantChain)
			}
		})
	}
}
