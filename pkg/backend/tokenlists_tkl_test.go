//go:build tkl

package backend

import "testing"

func TestWalletConfigOnLoginAccountWithNim(t *testing.T) { testWalletConfigOnLoginAccount(t, true) }
