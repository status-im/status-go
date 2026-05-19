package dnsname

import (
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

func TestEncode(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"", "00"},
		{"eth", "0365746800"},
		{"alice.eth", "05616c6963650365746800"},
		{"sub.alice.eth", "0373756205616c6963650365746800"},
	}
	for _, tc := range cases {
		got, err := Encode(tc.name)
		if err != nil {
			t.Fatalf("Encode(%q): %v", tc.name, err)
		}
		if hex.EncodeToString(got) != tc.want {
			t.Errorf("Encode(%q) = %x, want %s", tc.name, got, tc.want)
		}
	}
}

func TestEncode_EmptyLabel(t *testing.T) {
	_, err := Encode("a..b")
	if !errors.Is(err, errEmptyLabel) {
		t.Fatalf("expected errEmptyLabel, got %v", err)
	}
}

func TestEncode_LabelTooLong(t *testing.T) {
	long := strings.Repeat("a", 64)
	_, err := Encode(long + ".eth")
	if !errors.Is(err, errLabelTooLong) {
		t.Fatalf("expected errLabelTooLong, got %v", err)
	}
}
