package requests

import (
	"crypto/ecdsa"
	"errors"
	"strings"

	"github.com/status-im/status-go/eth-node/crypto"
)

var (
	ErrLoginInvalidKeyUID                   = errors.New("login: invalid key-uid")
	ErrLoginInvalidKeycardWhisperPrivateKey = errors.New("login: invalid keycard whisper private key")
)

type Login struct {
	Password string `json:"password"`
	KeyUID   string `json:"keyUid"`

	KdfIterations         int    `json:"kdfIterations"` // FIXME: KdfIterations should be loaded from multiaccounts db.
	RuntimeLogLevel       string `json:"runtimeLogLevel"`
	WakuV2Nameserver      string `json:"wakuV2Nameserver"`
	BandwidthStatsEnabled bool   `json:"bandwidthStatsEnabled"`
	
	KeycardWhisperPrivateKey string `json:"keycardWhisperPrivateKey"`

	// Mnemonic allows to log in to an account when password is lost.
	// This is needed for the "Lost keycard -> Start using without keycard" flow, when a keycard account database
	// exists locally, but now the keycard is lost. In this case client is responsible for calling
	// `convertToRegularAccount` after a successful login. This could be improved in the future.
	// When non-empty, mnemonic is used to generate required keypairs and:
	// - Password is ignored and replaced with encryption private key
	// - KeycardWhisperPrivateKey is ignored and replaced with chat private key
	Mnemonic string `json:"mnemonic"`

	WalletSecretsConfig
}

func (c *Login) Validate() error {
	if c.KeyUID == "" {
		return ErrLoginInvalidKeyUID
	}

	if c.KeycardWhisperPrivateKey != "" {
		_, err := parsePrivateKey(c.KeycardWhisperPrivateKey)
		if err != nil {
			return ErrLoginInvalidKeycardWhisperPrivateKey
		}
	}

	return nil
}

func (c *Login) ChatPrivateKey() *ecdsa.PrivateKey {
	// Skip error check, as it's already validated in Validate
	privateKey, _ := parsePrivateKey(c.KeycardWhisperPrivateKey)
	return privateKey
}

func parsePrivateKey(privateKeyHex string) (*ecdsa.PrivateKey, error) {
	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")
	return crypto.HexToECDSA(privateKeyHex)
}
