package transactions

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/eth-node/types"
)

type UnlockNonceFunc func(inc bool, n uint64)

type Nonce struct {
	addrLock   *AddrLocker
	localNonce map[uint64]*sync.Map
}

func NewNonce() *Nonce {
	return &Nonce{
		addrLock:   &AddrLocker{},
		localNonce: make(map[uint64]*sync.Map),
	}
}

func (n *Nonce) Next(rpcWrapper *rpcWrapper, from types.Address) (uint64, UnlockNonceFunc, error) {
	n.addrLock.LockAddr(from)
	current, err := n.GetCurrent(rpcWrapper, from)
	if err != nil {
		return 0, nil, err
	}

	unlock := func(inc bool, nonce uint64) {
		if inc {
			if _, ok := n.localNonce[rpcWrapper.chainID]; !ok {
				n.localNonce[rpcWrapper.chainID] = &sync.Map{}
			}

			n.localNonce[rpcWrapper.chainID].Store(from, nonce+1)
		}
		n.addrLock.UnlockAddr(from)
	}

	return current, unlock, nil
}

func (n *Nonce) GetCurrent(rpcWrapper *rpcWrapper, from types.Address) (uint64, error) {
	var (
		localNonce  uint64
		remoteNonce uint64
	)
	if _, ok := n.localNonce[rpcWrapper.chainID]; !ok {
		n.localNonce[rpcWrapper.chainID] = &sync.Map{}
	}

	// get the local nonce
	if val, ok := n.localNonce[rpcWrapper.chainID].Load(from); ok {
		localNonce = val.(uint64)
	}

	// get the remote nonce
	ctx := context.Background()
	remoteNonce, err := rpcWrapper.PendingNonceAt(ctx, common.Address(from))
	if err != nil {
		return 0, err
	}

	// if upstream node returned nonce higher than ours we will use it, as it probably means
	// that another client was used for sending transactions
	if remoteNonce > localNonce {
		return remoteNonce, nil
	}
	return localNonce, nil
}
