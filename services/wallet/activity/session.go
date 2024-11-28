package activity

import (
	"strconv"
	"sync"

	eth "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/transfer"
)

const nilStr = "nil"

type EntryIdentity struct {
	payloadType PayloadType
	transaction *transfer.TransactionIdentity
	id          common.MultiTransactionIDType
}

func (e EntryIdentity) same(a EntryIdentity) bool {
	return a.payloadType == e.payloadType &&
		((a.transaction == nil && e.transaction == nil) ||
			(a.transaction.ChainID == e.transaction.ChainID &&
				a.transaction.Hash == e.transaction.Hash &&
				a.transaction.Address == e.transaction.Address)) &&
		a.id == e.id
}

func (e EntryIdentity) key() string {
	txID := nilStr
	if e.transaction != nil {
		txID = strconv.FormatUint(uint64(e.transaction.ChainID), 10) + e.transaction.Hash.Hex() + e.transaction.Address.Hex()
	}
	return strconv.Itoa(e.payloadType) + txID + strconv.FormatInt(int64(e.id), 16)
}

type SessionID int32

// Session stores state related to a filter session
type Session struct {
	id      SessionID
	version Version

	// Filter info
	//
	addresses []eth.Address
	chainIDs  []common.ChainID
	filter    Filter

	// model is a mirror of the data model presentation has (sent by EventActivityFilteringDone)
	model []EntryIdentity
	// noFilterModel is a mirror of the data model presentation has when filter is empty
	noFilterModel map[string]EntryIdentity
	// new holds the new entries until user requests update by calling ResetFilterSession
	new []EntryIdentity

	mu *sync.RWMutex
}

func (s *Session) getFullFilterParams() fullFilterParams {
	return fullFilterParams{
		sessionID: s.id,
		version:   s.version,
		addresses: s.addresses,
		chainIDs:  s.chainIDs,
		filter:    s.filter,
	}
}
