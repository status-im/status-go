package walletconnect

import (
	"errors"
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

func Test_isValidNamespaceName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid_eip155", "eip155", true},
		{"valid_cosmos", "cosmos", true},
		{"valid_with_dash", "eip-155", true},
		{"valid_3_chars", "abc", true},
		{"valid_8_chars", "abcdefgh", true},
		{"valid_with_numbers", "eip155", true},
		{"invalid_empty", "", false},
		{"invalid_too_short", "ab", false},
		{"invalid_too_long", "abcdefghi", false},
		{"invalid_uppercase", "EIP155", false},
		{"invalid_special_chars", "eip_155", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidNamespaceName(tt.input)
			if got != tt.expected {
				t.Errorf("isValidNamespaceName(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func Test_JSONProxyType_UnmarshalJSON(t *testing.T) {
	t.Run("nil_transform_returns_error", func(t *testing.T) {
		var result string
		proxy := &JSONProxyType{
			target:    &result,
			transform: nil,
		}
		err := proxy.UnmarshalJSON([]byte(`"hello"`))
		if err == nil {
			t.Error("expected error when transform is nil, got nil")
			return
		}
		if err.Error() != "transform function is not set" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("transform_error_propagates", func(t *testing.T) {
		transformErr := errors.New("transform failed")
		var result string
		proxy := &JSONProxyType{
			target: &result,
			transform: func(input []byte) ([]byte, error) {
				return nil, transformErr
			},
		}
		err := proxy.UnmarshalJSON([]byte(`"hello"`))
		if err != transformErr {
			t.Errorf("expected transform error, got %v", err)
		}
	})

	t.Run("successful_transform_and_unmarshal", func(t *testing.T) {
		var result string
		proxy := &JSONProxyType{
			target: &result,
			transform: func(input []byte) ([]byte, error) {
				return input, nil
			},
		}
		err := proxy.UnmarshalJSON([]byte(`"hello"`))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}
		if result != "hello" {
			t.Errorf("expected result 'hello', got %q", result)
		}
	})

	t.Run("transform_modifies_json", func(t *testing.T) {
		var result struct {
			Value int `json:"value"`
		}
		proxy := &JSONProxyType{
			target: &result,
			transform: func(input []byte) ([]byte, error) {
				return []byte(`{"value": 42}`), nil
			},
		}
		err := proxy.UnmarshalJSON([]byte(`{"value": 1}`))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}
		if result.Value != 42 {
			t.Errorf("expected value 42, got %d", result.Value)
		}
	})

	t.Run("invalid_json_after_transform", func(t *testing.T) {
		var result struct {
			Value int `json:"value"`
		}
		proxy := &JSONProxyType{
			target: &result,
			transform: func(input []byte) ([]byte, error) {
				return []byte(`{invalid`), nil
			},
		}
		err := proxy.UnmarshalJSON([]byte(`{}`))
		if err == nil {
			t.Error("expected error for invalid JSON, got nil")
		}
	})
}
