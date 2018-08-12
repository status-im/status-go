package doubleratchet

import "fmt"

// Session of the party involved in the Double Ratchet Algorithm.
type Session interface {
	// RatchetEncrypt performs a symmetric-key ratchet step, then AEAD-encrypts the message with
	// the resulting message key.
	RatchetEncrypt(plaintext, associatedData []byte) (Message, error)

	// RatchetDecrypt is called to AEAD-decrypt messages.
	RatchetDecrypt(m Message, associatedData []byte) ([]byte, error)
}

type sessionState struct {
	id []byte
	State
	storage SessionStorage
}

// New creates session with the shared key.
func New(id []byte, sharedKey Key, keyPair DHPair, storage SessionStorage, opts ...option) (Session, error) {
	state, err := newState(sharedKey, opts...)
	if err != nil {
		return nil, err
	}
	state.DHs = keyPair

	session := &sessionState{id: id, State: state, storage: storage}

	return session, session.store()
}

// NewWithRemoteKey creates session with the shared key and public key of the other party.
func NewWithRemoteKey(id []byte, sharedKey, remoteKey Key, storage SessionStorage, opts ...option) (Session, error) {
	state, err := newState(sharedKey, opts...)
	if err != nil {
		return nil, err
	}
	state.DHs, err = state.Crypto.GenerateDH()
	if err != nil {
		return nil, fmt.Errorf("can't generate key pair: %s", err)
	}
	state.DHr = remoteKey
	state.SendCh, _ = state.RootCh.step(state.Crypto.DH(state.DHs, state.DHr))

	session := &sessionState{id: id, State: state, storage: storage}

	return session, session.store()
}

// Load a session from a SessionStorage implementation and apply options.
func Load(id []byte, store SessionStorage, opts ...option) (Session, error) {
	state, err := store.Load(id)
	if err != nil {
		return nil, err
	}

	if state == nil {
		return nil, nil
	}

	if err = state.applyOptions(opts); err != nil {
		return nil, err
	}

	s := &sessionState{id: id, State: *state}
	s.storage = store

	return s, nil
}

func (s *sessionState) store() error {
	if s.storage != nil {
		err := s.storage.Save(s.id, &s.State)
		if err != nil {
			return err
		}
	}
	return nil
}

// RatchetEncrypt performs a symmetric-key ratchet step, then encrypts the message with
// the resulting message key.
func (s *sessionState) RatchetEncrypt(plaintext, ad []byte) (Message, error) {
	var (
		h = MessageHeader{
			DH: s.DHs.PublicKey(),
			N:  s.SendCh.N,
			PN: s.PN,
		}
		mk = s.SendCh.step()
	)
	ct := s.Crypto.Encrypt(mk, plaintext, append(ad, h.Encode()...))

	// Store state
	if err := s.store(); err != nil {
		return Message{}, err
	}

	return Message{h, ct}, nil
}

// RatchetDecrypt is called to decrypt messages.
func (s *sessionState) RatchetDecrypt(m Message, ad []byte) ([]byte, error) {
	// Is the message one of the skipped?
	mk, ok, err := s.MkSkipped.Get(m.Header.DH, uint(m.Header.N))
	if err != nil {
		return nil, err
	}

	if ok {
		plaintext, err := s.Crypto.Decrypt(mk, m.Ciphertext, append(ad, m.Header.Encode()...))
		if err != nil {
			return nil, fmt.Errorf("can't decrypt skipped message: %s", err)
		}
		_ = s.MkSkipped.DeleteMk(m.Header.DH, uint(m.Header.N))
		return plaintext, nil
	}

	var (
		// All changes must be applied on a different session object, so that this session won't be modified nor left in a dirty session.
		sc = s.State

		skippedKeys1 []skippedKey
		skippedKeys2 []skippedKey
	)

	// Is there a new ratchet key?
	isDHStepped := false
	if m.Header.DH != sc.DHr {
		if skippedKeys1, err = sc.skipMessageKeys(sc.DHr, uint(m.Header.PN)); err != nil {
			return nil, fmt.Errorf("can't skip previous chain message keys: %s", err)
		}
		if err = sc.dhRatchet(m.Header); err != nil {
			return nil, fmt.Errorf("can't perform ratchet step: %s", err)
		}
		isDHStepped = true
	}

	// After all, update the current chain.
	if skippedKeys2, err = sc.skipMessageKeys(sc.DHr, uint(m.Header.N)); err != nil {
		return nil, fmt.Errorf("can't skip current chain message keys: %s", err)
	}
	mk = sc.RecvCh.step()
	plaintext, err := s.Crypto.Decrypt(mk, m.Ciphertext, append(ad, m.Header.Encode()...))
	if err != nil {
		return nil, fmt.Errorf("can't decrypt: %s", err)
	}

	// Apply changes.
	if err := s.applyChanges(sc, append(skippedKeys1, skippedKeys2...)); err != nil {
		return nil, err
	}

	if isDHStepped {
		err = s.deleteSkippedKeys(s.DHr)
		if err != nil {
			return nil, err
		}
	}

	// Store state
	if err := s.store(); err != nil {
		return nil, err
	}

	return plaintext, nil
}
