//go:build !tkl

package token

import (
	"errors"
	"time"

	"github.com/status-im/status-go/pkg/services/wallet/token/tokenlist"
)

func selectTokenListsManager(m *Manager, chains []uint64, last time.Time, refresh, check time.Duration, useNim bool) (tokenlist.Catalogue, error) {
	if useNim {
		return nil, errors.New("TokenListsUseNim requires a build with the tkl tag")
	}
	return setUpTokenListsManager(m, m.walletDB, chains, last, refresh, check)
}
