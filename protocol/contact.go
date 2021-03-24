package protocol

import (
	"crypto/ecdsa"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/protocol/identity/alias"
	"github.com/status-im/status-go/protocol/identity/identicon"
)

func (c *Contact) CanonicalName() string {
	if c.LocalNickname != "" {
		return c.LocalNickname
	}

	if c.ENSVerified {
		return c.Name
	}

	return c.Alias
}

func (c *Contact) CanonicalImage() string {
	return c.Identicon
}

// Contact has information about a "Contact"
type Contact struct {
	// ID of the contact. It's a hex-encoded public key (prefixed with 0x).
	ID string `json:"id"`
	// Ethereum address of the contact
	Address string `json:"address,omitempty"`
	// ENS name of contact
	Name string `json:"name,omitempty"`
	// EnsVerified whether we verified the name of the contact
	ENSVerified bool `json:"ensVerified"`
	// Generated username name of the contact
	Alias string `json:"alias,omitempty"`
	// Identicon generated from public key
	Identicon string `json:"identicon"`
	// LastUpdated is the last time we received an update from the contact
	// updates should be discarded if last updated is less than the one stored
	LastUpdated uint64 `json:"lastUpdated"`

	LocalNickname string `json:"localNickname,omitempty"`

	Images map[string]images.IdentityImage `json:"images"`

	Added   bool `json:"added"`
	Blocked bool `json:"blocked"`

	IsSyncing bool
	Removed   bool
}

func (c Contact) PublicKey() (*ecdsa.PublicKey, error) {
	b, err := types.DecodeHex(c.ID)
	if err != nil {
		return nil, err
	}
	return crypto.UnmarshalPubkey(b)
}

func (c Contact) IsAdded() bool {
	return c.Added
}

func (c Contact) IsBlocked() bool {
	return c.Blocked
}

func (c *Contact) Block() {
	c.Blocked = true
}

func (c *Contact) Unblock() {
	c.Blocked = false
	c.Added = true
}

func (c *Contact) Remove() {
	c.Added = false
	c.Removed = true
}

func buildContactFromPkString(pkString string) (*Contact, error) {
	publicKeyBytes, err := types.DecodeHex(pkString)
	if err != nil {
		return nil, err
	}

	publicKey, err := crypto.UnmarshalPubkey(publicKeyBytes)
	if err != nil {
		return nil, err
	}

	return buildContact(pkString, publicKey)
}

func BuildContactFromPublicKey(publicKey *ecdsa.PublicKey) (*Contact, error) {
	id := types.EncodeHex(crypto.FromECDSAPub(publicKey))
	return buildContact(id, publicKey)
}

func buildContact(publicKeyString string, publicKey *ecdsa.PublicKey) (*Contact, error) {
	newIdenticon, err := identicon.GenerateBase64(publicKeyString)
	if err != nil {
		return nil, err
	}

	contact := &Contact{
		ID:        publicKeyString,
		Alias:     alias.GenerateFromPublicKey(publicKey),
		Identicon: newIdenticon,
	}

	return contact, nil
}

func contactIDFromPublicKey(key *ecdsa.PublicKey) string {
	return types.EncodeHex(crypto.FromECDSAPub(key))
}
