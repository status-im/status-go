package statusproto

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	contactBlocked         = "contact/blocked"
	contactAdded           = "contact/added"
	contactRequestReceived = "contact/request-received"
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
	Address string `json:"address"`
	// Name of contact
	Name string `json:"name"`
	// Photo is the base64 encoded photo
	Photo string `json:"photoPath"`
	// LastUpdated is the last time we received an update from the contact
	// updates should be discarded if last updated is less than the one stored
	LastUpdated int64 `json:"lastUpdated"`
	// SystemTags contains information about whether we blocked/added/have been
	// added.
	SystemTags []string `json:"systemTags"`

	DeviceInfo    []ContactDeviceInfo `json:"deviceInfo"`
	TributeToTalk string              `json:"tributeToTalk"`
}

func (c Contact) PublicKey() (*ecdsa.PublicKey, error) {
	b, err := hexutil.Decode(c.ID)
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

// existsInStringSlice checks if a string is in a set.
func existsInStringSlice(set []string, find string) bool {
	for _, s := range set {
		if s == find {
			return true
		}
	}
	return false
}
