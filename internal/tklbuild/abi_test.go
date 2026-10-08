package tklbuild

import (
	"testing"

	"github.com/status-im/nim-token-lists/go/tkl"
)

func TestPublicLibraryABI(t *testing.T) {
	if got := tkl.ABIVersion(); got != 2 {
		t.Fatalf("unexpected token-library ABI version: %d", got)
	}
}
