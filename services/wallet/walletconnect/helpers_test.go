package walletconnect

import "testing"

func Test_parseCaip2ChainID(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name    string
		args    args
		want    uint64
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				str: "eip155:5",
			},
			want:    5,
			wantErr: false,
		},
		{
			name: "invalid_number",
			args: args{
				str: "eip155:5a",
			},
			want:    0,
			wantErr: true,
		},
		{
			name: "invalid_caip2_too_many",
			args: args{
				str: "eip155:1:5",
			},
			want:    0,
			wantErr: true,
		},
		{
			name: "invalid_caip2_not_enough",
			args: args{
				str: "eip1551",
			},
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCaip2ChainID(tt.args.str)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCaip2ChainID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseCaip2ChainID() = %v, want %v", got, tt.want)
			}
		})
	}
}
