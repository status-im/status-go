package doubleratchet

import "fmt"

// Session of the party involved in the Double Ratchet Algorithm.
type Session interface {
	// RatchetEncrypt performs a symmetric-key ratchet step, then AEAD-encrypts the message with
	// the resulting message key.
	RatchetEncrypt(plaintext, associatedData []byte) Message

	// RatchetDecrypt is called to AEAD-decrypt messages.
	RatchetDecrypt(m Message, associatedData []byte) ([]byte, error)
}

type session struct {
	state
}

// New creates session with the shared key.
func New(sharedKey Key, keyPair DHPair, opts ...option) (Session, error) {
	state, err := newState(sharedKey, opts...)
	if err != nil {
		return nil, err
	}
	state.DHs = keyPair
	return &session{state}, nil
}

// NewWithRemoteKey creates session with the shared key and public key of the other party.
func NewWithRemoteKey(sharedKey, remoteKey Key, opts ...option) (Session, error) {
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
	return &session{state}, nil
}

// RatchetEncrypt performs a symmetric-key ratchet step, then encrypts the message with
// the resulting message key.
func (s *session) RatchetEncrypt(plaintext, ad []byte) Message {
	var (
		h = MessageHeader{
			DH: s.DHs.PublicKey(),
			N:  s.SendCh.N,
			PN: s.PN,
		}
		mk = s.SendCh.step()
	)
	ct := s.Crypto.Encrypt(mk, plaintext, append(ad, h.Encode()...))
	return Message{h, ct}
}

// RatchetDecrypt is called to decrypt messages.
func (s *session) RatchetDecrypt(m Message, ad []byte) ([]byte, error) {
	// Is the message one of the skipped?
	if mk, ok := s.MkSkipped.Get(m.Header.DH, uint(m.Header.N)); ok {
		plaintext, err := s.Crypto.Decrypt(mk, m.Ciphertext, append(ad, m.Header.Encode()...))
		if err != nil {
			return nil, fmt.Errorf("can't decrypt skipped message: %s", err)
		}
		s.MkSkipped.DeleteMk(m.Header.DH, uint(m.Header.N))
		return plaintext, nil
	}

	var (
		// All changes must be applied on a different session object, so that this session won't be modified nor left in a dirty session.
		sc state = s.state

		skippedKeys1 []skippedKey
		skippedKeys2 []skippedKey
		err          error
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
	mk := sc.RecvCh.step()
	plaintext, err := s.Crypto.Decrypt(mk, m.Ciphertext, append(ad, m.Header.Encode()...))
	if err != nil {
		return nil, fmt.Errorf("can't decrypt: %s", err)
	}

	// Apply changes.
	s.applyChanges(sc, append(skippedKeys1, skippedKeys2...))
	if isDHStepped {
		s.deleteSkippedKeys(s.DHr)
	}

	return plaintext, nil
}
