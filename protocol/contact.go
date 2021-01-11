package protocol

import (
	"crypto/ecdsa"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/protocol/identity/alias"
	"github.com/status-im/status-go/protocol/identity/identicon"
)

const (
	contactBlocked         = ":contact/blocked"
	contactAdded           = ":contact/added"
	contactRequestReceived = ":contact/request-received"
)

// ContactDeviceInfo is a struct containing information about a particular device owned by a contact
type ContactDeviceInfo struct {
	// The installation id of the device
	InstallationID string `json:"id"`
	// Timestamp represents the last time we received this info
	Timestamp int64 `json:"timestamp"`
	// FCMToken is to be used for push notifications
	FCMToken string `json:"fcmToken"`
}

// Contact has information about a "Contact". A contact is not necessarily one
// that we added or added us, that's based on SystemTags.
type Contact struct {
	// ID of the contact. It's a hex-encoded public key (prefixed with 0x).
	ID string `json:"id"`
	// Ethereum address of the contact
	Address string `json:"address,omitempty"`
	// ENS name of contact
	Name string `json:"name,omitempty"`
	// EnsVerified whether we verified the name of the contact
	ENSVerified bool `json:"ensVerified"`
	// EnsVerifiedAt the time we last verified the name
	ENSVerifiedAt uint64 `json:"ensVerifiedAt"`
	// LastENSClockValue is the last clock value of when we
	// received an ENS name for the user
	LastENSClockValue uint64 `json:"lastENSClockValue"`
	// ENSVerificationRetries is how many times we retried the ENS
	ENSVerificationRetries uint64 `json:"ensVerificationRetries"`
	// Generated username name of the contact
	Alias string `json:"alias,omitempty"`
	// Identicon generated from public key
	Identicon string `json:"identicon"`
	// LastUpdated is the last time we received an update from the contact
	// updates should be discarded if last updated is less than the one stored
	LastUpdated uint64 `json:"lastUpdated"`
	// SystemTags contains information about whether we blocked/added/have been
	// added.
	SystemTags []string `json:"systemTags"`

	DeviceInfo    []ContactDeviceInfo `json:"deviceInfo"`
	TributeToTalk string              `json:"tributeToTalk,omitempty"`
	LocalNickname string              `json:"localNickname,omitempty"`

	Images map[string]images.IdentityImage `json:"images"`
}

func (c Contact) PublicKey() (*ecdsa.PublicKey, error) {
	b, err := types.DecodeHex(c.ID)
	if err != nil {
		return nil, err
	}
	return crypto.UnmarshalPubkey(b)
}

func (c Contact) IsAdded() bool {
	return existsInStringSlice(c.SystemTags, contactAdded)
}

func (c Contact) HasBeenAdded() bool {
	return existsInStringSlice(c.SystemTags, contactRequestReceived)
}

func (c Contact) IsBlocked() bool {
	return existsInStringSlice(c.SystemTags, contactBlocked)
}

func (c *Contact) Remove() {
	var newSystemTags []string
	// Remove the newSystemTags system-tag, so that the contact is
	// not considered "added" anymore
	for _, tag := range newSystemTags {
		if tag != contactAdded {
			newSystemTags = append(newSystemTags, tag)
		}
	}
	c.SystemTags = newSystemTags
}

func (c *Contact) ResetENSVerification(clock uint64, name string) {
	c.ENSVerifiedAt = 0
	c.ENSVerified = false
	c.ENSVerificationRetries = 0
	c.LastENSClockValue = clock
	c.Name = name
}

// existsInStringSlice checks if a string is in a set.
func existsInStringSlice(set []string, find string) bool {
	for _, s := range set {
		if s == find {
			return true
		}
	}
	return false
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

func buildContactFromPublicKey(publicKey *ecdsa.PublicKey) (*Contact, error) {
	id := types.EncodeHex(crypto.FromECDSAPub(publicKey))
	return buildContact(id, publicKey)
}

func buildContact(publicKeyString string, publicKey *ecdsa.PublicKey) (*Contact, error) {
	identicon, err := identicon.GenerateBase64(publicKeyString)
	if err != nil {
		return nil, err
	}

	contact := &Contact{
		ID:        publicKeyString,
		Alias:     alias.GenerateFromPublicKey(publicKey),
		Identicon: identicon,
	}

	return contact, nil
}

// HasCustomFields returns whether the the contact has any field that is valuable
// to the client other than the computed name/image
func (c Contact) HasCustomFields() bool {
	return c.IsAdded() || c.HasBeenAdded() || c.IsBlocked() || c.ENSVerified || c.LocalNickname != "" || len(c.Images) != 0
}

func contactIDFromPublicKey(key *ecdsa.PublicKey) string {
	return types.EncodeHex(crypto.FromECDSAPub(key))

}
