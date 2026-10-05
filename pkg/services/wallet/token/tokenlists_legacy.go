//go:build !tkl

package token

import (
	"errors"
	"time"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/manager"
)

func selectTokenListsManager(m *Manager, chains []uint64, last time.Time, refresh, check time.Duration, useNim bool) (manager.Manager, error) {
	if useNim {
		return nil, errors.New("TokenListsUseNim requires a build with the tkl tag")
	}
	return setUpTokenListsManager(m, m.walletDB, chains, last, refresh, check)
}
