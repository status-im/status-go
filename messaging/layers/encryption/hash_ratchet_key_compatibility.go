package encryption

import (
	"errors"

	"github.com/status-im/status-go/crypto"
)

type HashRatchetKeyCompatibility struct {
	GroupID   []byte
	keyID     []byte
	Timestamp uint64
	Key       []byte
}

func (h *HashRatchetKeyCompatibility) DeprecatedKeyID() uint32 {
	return uint32(h.Timestamp)
}

func (h *HashRatchetKeyCompatibility) IsOldFormat() bool {
	return len(h.keyID) == 0 && len(h.Key) == 0
}

func (h *HashRatchetKeyCompatibility) GetKeyID() ([]byte, error) {
	if len(h.keyID) != 0 {
		return h.keyID, nil
	}

	if len(h.GroupID) == 0 || h.Timestamp == 0 || len(h.Key) == 0 {
		return nil, errors.New("could not create key")
	}

	return generateHashRatchetKeyID(h.GroupID, h.Timestamp, h.Key), nil
}

func (h *HashRatchetKeyCompatibility) GenerateNext() (*HashRatchetKeyCompatibility, error) {

	ratchet := &HashRatchetKeyCompatibility{
		GroupID: h.GroupID,
	}

	// Randomly generate a hash ratchet key
	hrKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	hrKeyBytes := crypto.FromECDSA(hrKey)

	if err != nil {
		return nil, err
	}

	currentTime := GetCurrentTime()
	if h.Timestamp < currentTime {
		ratchet.Timestamp = bumpKeyID(currentTime)
	} else {
		ratchet.Timestamp = h.Timestamp + 1
	}

	ratchet.Key = hrKeyBytes

	_, err = ratchet.GetKeyID()
	if err != nil {
		return nil, err
	}

	return ratchet, nil
}
