package transferdetector_test

import (
	"math/big"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/status-im/go-wallet-sdk/pkg/balance/refresh"
	"github.com/status-im/go-wallet-sdk/pkg/eventlog"

	"github.com/status-im/status-go/pkg/services/wallet/transferdetector"
)

var (
	account  = common.Address{0x11}
	usdc     = common.Address{0x05}
	spam     = common.Address{0x5b}
	stranger = common.Address{0x99}
)

type listedTokens []common.Address

func (l listedTokens) GetTokenContractAddresses(uint64) ([]common.Address, error) {
	return l, nil
}

func transferLog(contract, from, to common.Address) eventlog.Event {
	return eventlog.ParseLog(gethtypes.Log{
		Address: contract,
		Topics:  []common.Hash{eventlog.ERC20TransferID, common.BytesToHash(from[:]), common.BytesToHash(to[:])},
		Data:    common.BigToHash(big.NewInt(1)).Bytes(),
	})[0]
}

type emitted struct {
	mu     sync.Mutex
	events []refresh.Event
}

func (e *emitted) emit(ev refresh.Event) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, ev)
}

func (e *emitted) take() []refresh.Event {
	synctest.Wait()
	e.mu.Lock()
	defer e.mu.Unlock()
	events := e.events
	e.events = nil
	return events
}

func TestChangeSource_TellsTransfersOfKnownTokensAndNativeTransferLogs(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newBlocksHarness(t)
		source := transferdetector.NewChangeSource(h.controller, listedTokens{usdc}, zap.NewNop())
		var e emitted
		stop := source.Watch(e.emit)
		defer stop()
		h.readAt(100)
		h.start()
		defer h.controller.Stop()

		h.mine(
			transferLog(usdc, stranger, account),
			transferLog(spam, stranger, account),
			transferLog(refresh.NativeTransferLogAddress, stranger, account),
		)
		h.readAt(110)
		source.Scan()
		h.waitDebounce()

		require.Equal(t, []refresh.Event{
			refresh.TransferDetected{Pair: refresh.Pair{Account: account, Chain: 1}, Contracts: []common.Address{usdc}, Native: true},
		}, e.take())
	})
}

func TestChangeSource_TellsARangeItFailedToLookAtAsADetectionGap(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newBlocksHarness(t)
		source := transferdetector.NewChangeSource(h.controller, listedTokens{usdc}, zap.NewNop())
		var e emitted
		stop := source.Watch(e.emit)
		defer stop()
		h.readAt(100)
		h.start()
		defer h.controller.Stop()

		h.setFailing(true)
		h.readAt(110)
		source.Scan()
		h.waitDebounce()

		require.Equal(t, []refresh.Event{refresh.DetectionGap{Chain: 1}}, e.take())
	})
}

func TestChangeSource_StopsTellingWhenStopped(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newBlocksHarness(t)
		source := transferdetector.NewChangeSource(h.controller, listedTokens{usdc}, zap.NewNop())
		var e emitted
		stop := source.Watch(e.emit)
		h.readAt(100)
		h.start()
		defer h.controller.Stop()

		stop()
		h.mine(transferLog(usdc, stranger, account))
		h.readAt(110)
		h.scan()

		require.Empty(t, e.take())
	})
}
