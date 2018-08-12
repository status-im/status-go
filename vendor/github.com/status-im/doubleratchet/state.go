package doubleratchet

// TODO: During each DH ratchet step a new ratchet key pair and sending chain are generated.
// As the sending chain is not needed right away, these steps could be deferred until the party
// is about to send a new message.

import (
	"fmt"
)

// The double ratchet state.
type State struct {
	Crypto Crypto

	// DH Ratchet public key (the remote key).
	DHr Key

	// DH Ratchet key pair (the self ratchet key).
	DHs DHPair

	// Symmetric ratchet root chain.
	RootCh kdfRootChain

	// Symmetric ratchet sending and receiving chains.
	SendCh, RecvCh kdfChain

	// Number of messages in previous sending chain.
	PN uint32

	// Dictionary of skipped-over message keys, indexed by ratchet public key or header key
	// and message number.
	MkSkipped KeysStorage

	// The maximum number of message keys that can be skipped in a single chain.
	// WithMaxSkip should be set high enough to tolerate routine lost or delayed messages,
	// but low enough that a malicious sender can't trigger excessive recipient computation.
	MaxSkip uint

	// Receiving header key and next header key. Only used for header encryption.
	HKr, NHKr Key

	// Sending header key and next header key. Only used for header encryption.
	HKs, NHKs Key

	// Number of ratchet steps after which all skipped message keys for that key will be deleted.
	MaxKeep uint

	// The number of the current ratchet step.
	Step uint

	// Which key for the receiving chain was used at the specified step.
	DeleteKeys map[uint]Key
}

func DefaultState(sharedKey Key) State {
	c := DefaultCrypto{}

	return State{
		DHs:    dhPair{},
		Crypto: c,
		RootCh: kdfRootChain{CK: sharedKey, Crypto: c},
		// Populate CKs and CKr with sharedKey so that both parties could send and receive
		// messages from the very beginning.
		SendCh:     kdfChain{CK: sharedKey, Crypto: c},
		RecvCh:     kdfChain{CK: sharedKey, Crypto: c},
		MkSkipped:  &KeysStorageInMemory{},
		MaxSkip:    1000,
		MaxKeep:    100,
		DeleteKeys: make(map[uint]Key),
	}
}

func (s *State) applyOptions(opts []option) error {
	for i := range opts {
		if err := opts[i](s); err != nil {
			return fmt.Errorf("failed to apply option: %s", err)
		}
	}
	return nil
}

func newState(sharedKey Key, opts ...option) (State, error) {
	if sharedKey == [32]byte{} {
		return State{}, fmt.Errorf("sharedKey mustn't be empty")
	}

	s := DefaultState(sharedKey)
	if err := s.applyOptions(opts); err != nil {
		return State{}, err
	}

	return s, nil
}

// dhRatchet performs a single ratchet step.
func (s *State) dhRatchet(m MessageHeader) error {
	s.PN = s.SendCh.N
	s.DHr = m.DH
	s.HKs = s.NHKs
	s.HKr = s.NHKr
	s.RecvCh, s.NHKr = s.RootCh.step(s.Crypto.DH(s.DHs, s.DHr))
	var err error
	s.DHs, err = s.Crypto.GenerateDH()
	if err != nil {
		return fmt.Errorf("failed to generate dh pair: %s", err)
	}
	s.SendCh, s.NHKs = s.RootCh.step(s.Crypto.DH(s.DHs, s.DHr))
	return nil
}

type skippedKey struct {
	key Key
	nr  uint
	mk  Key
}

// skipMessageKeys skips message keys in the current receiving chain.
func (s *State) skipMessageKeys(key Key, until uint) ([]skippedKey, error) {
	if until < uint(s.RecvCh.N) {
		return nil, fmt.Errorf("bad until: probably an out-of-order message that was deleted")
	}
	nSkipped, err := s.MkSkipped.Count(key)
	if err != nil {
		return nil, err
	}

	if until-uint(s.RecvCh.N)+nSkipped > s.MaxSkip {
		return nil, fmt.Errorf("too many messages")
	}
	skipped := []skippedKey{}
	for uint(s.RecvCh.N) < until {
		mk := s.RecvCh.step()
		skipped = append(skipped, skippedKey{
			key: key,
			nr:  uint(s.RecvCh.N - 1),
			mk:  mk,
		})
	}
	return skipped, nil
}

func (s *State) applyChanges(sc State, skipped []skippedKey) error {
	*s = sc
	for _, skipped := range skipped {
		if err := s.MkSkipped.Put(skipped.key, skipped.nr, skipped.mk); err != nil {
			return err
		}
	}

	return nil
}

func (s *State) deleteSkippedKeys(key Key) error {

	s.DeleteKeys[s.Step] = key
	s.Step++
	if hk, ok := s.DeleteKeys[s.Step-s.MaxKeep]; ok {
		if err := s.MkSkipped.DeletePk(hk); err != nil {
			return err
		}

		delete(s.DeleteKeys, s.Step-s.MaxKeep)
	}
	return nil
}
