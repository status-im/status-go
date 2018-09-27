package globalplatform

import (
	"errors"
	"fmt"

	"github.com/status-im/status-go/smartcard/apdu"
	"github.com/status-im/status-go/smartcard/globalplatform/crypto"
)

type Session struct {
	keyProvider   *KeyProvider
	cardChallenge []byte
}

var errBadCryptogram = errors.New("bad card cryptogram")

func NewSession(cardKeys *KeyProvider, resp *apdu.Response, hostChallenge []byte) (*Session, error) {
	if resp.Sw == apdu.SwSecurityConditionNotSatisfied {
		return nil, apdu.NewErrBadResponse(resp.Sw, "security condition not satisfied")
	}

	if resp.Sw == apdu.SwAuthenticationMethodBlocked {
		return nil, apdu.NewErrBadResponse(resp.Sw, "authentication method blocked")
	}

	if len(resp.Data) != 28 {
		return nil, apdu.NewErrBadResponse(resp.Sw, fmt.Sprintf("bad data length, expected 28, got %d", len(resp.Data)))
	}

	cardChallenge := resp.Data[12:20]
	cardCryptogram := resp.Data[20:28]
	seq := resp.Data[12:14]

	sessionEncKey, err := crypto.DeriveKey(cardKeys.Enc(), seq, crypto.DerivationPurposeEnc)
	if err != nil {
		return nil, err
	}

	sessionMacKey, err := crypto.DeriveKey(cardKeys.Enc(), seq, crypto.DerivationPurposeMac)
	if err != nil {
		return nil, err
	}

	sessionKeys := NewKeyProvider(sessionEncKey, sessionMacKey)
	verified, err := crypto.VerifyCryptogram(sessionKeys.Enc(), hostChallenge, cardChallenge, cardCryptogram)
	if err != nil {
		return nil, err
	}

	if !verified {
		return nil, errBadCryptogram
	}

	s := &Session{
		keyProvider:   sessionKeys,
		cardChallenge: cardChallenge,
	}

	return s, nil
}
