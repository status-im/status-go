package protocol

import (
	"crypto/ecdsa"
	"encoding/json"

	"github.com/status-im/status-go/api/multiformat"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/identity"
	"github.com/status-im/status-go/protocol/identity/alias"
	"github.com/status-im/status-go/protocol/identity/identicon"
	"github.com/status-im/status-go/protocol/verification"
)

type ContactRequestState int

const (
	ContactRequestStateNone ContactRequestState = iota
	ContactRequestStateMutual
	ContactRequestStateSent
	ContactRequestStateReceived
	ContactRequestStateDismissed
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

func (c *Contact) CanonicalName() string {
	if c.LocalNickname != "" {
		return c.LocalNickname
	}

	if c.ENSVerified {
		return c.EnsName
	}

	return c.Alias
}

func (c *Contact) CanonicalImage(profilePicturesVisibility settings.ProfilePicturesVisibilityType) string {
	if profilePicturesVisibility == settings.ProfilePicturesVisibilityNone || (profilePicturesVisibility == settings.ProfilePicturesVisibilityContactsOnly && !c.Added) {
		return c.Identicon
	}

	if largeImage, ok := c.Images[images.LargeDimName]; ok {
		imageBase64, err := largeImage.GetDataURI()
		if err == nil {
			return imageBase64
		}
	}

	if thumbImage, ok := c.Images[images.SmallDimName]; ok {
		imageBase64, err := thumbImage.GetDataURI()
		if err == nil {
			return imageBase64
		}
	}

	return c.Identicon
}

type VerificationStatus int

const (
	VerificationStatusUNVERIFIED VerificationStatus = iota
	VerificationStatusVERIFYING
	VerificationStatusVERIFIED
)

// Contact has information about a "Contact"
type Contact struct {
	// ID of the contact. It's a hex-encoded public key (prefixed with 0x).
	ID string `json:"id"`
	// Ethereum address of the contact
	Address string `json:"address,omitempty"`
	// ENS name of contact
	EnsName string `json:"name,omitempty"`
	// EnsVerified whether we verified the name of the contact
	ENSVerified bool `json:"ensVerified"`
	// Generated username name of the contact
	Alias string `json:"alias,omitempty"`
	// Identicon generated from public key
	Identicon string `json:"identicon"`
	// LastUpdated is the last time we received an update from the contact
	// updates should be discarded if last updated is less than the one stored
	LastUpdated uint64 `json:"lastUpdated"`

	// LastUpdatedLocally is the last time we updated the contact locally
	LastUpdatedLocally uint64 `json:"lastUpdatedLocally"`

	LocalNickname string `json:"localNickname,omitempty"`

	// Display name of the contact
	DisplayName string `json:"displayName"`

	// Bio - description of the contact (tell us about yourself)
	Bio string `json:"bio"`

	SocialLinks identity.SocialLinks `json:"socialLinks"`

	Images map[string]images.IdentityImage `json:"images"`

	Added      bool `json:"added"`
	Blocked    bool `json:"blocked"`
	HasAddedUs bool `json:"hasAddedUs"`

	ContactRequestState ContactRequestState `json:"contactRequestState"`
	ContactRequestClock uint64              `json:"contactRequestClock"`

	IsSyncing bool
	Removed   bool

	VerificationStatus VerificationStatus       `json:"verificationStatus"`
	TrustStatus        verification.TrustStatus `json:"trustStatus"`
}

func (c Contact) IsVerified() bool {
	return c.VerificationStatus == VerificationStatusVERIFIED
}

func (c Contact) IsVerifying() bool {
	return c.VerificationStatus == VerificationStatusVERIFYING
}

func (c Contact) IsUnverified() bool {
	return c.VerificationStatus == VerificationStatusUNVERIFIED
}

func (c Contact) IsUntrustworthy() bool {
	return c.TrustStatus == verification.TrustStatusUNTRUSTWORTHY
}

func (c Contact) IsTrusted() bool {
	return c.TrustStatus == verification.TrustStatusTRUSTED
}

func (c Contact) PublicKey() (*ecdsa.PublicKey, error) {
	b, err := types.DecodeHex(c.ID)
	if err != nil {
		return nil, err
	}
	return crypto.UnmarshalPubkey(b)
}

func (c *Contact) Block() {
	c.Blocked = true
	c.Added = false
}

func (c *Contact) BlockDesktop() {
	c.Blocked = true
}

func (c *Contact) Unblock() {
	c.Blocked = false
}

func (c *Contact) Remove() {
	c.Added = false
	c.Removed = true
}

func (c *Contact) Add() {
	c.Added = true
	c.Removed = false
}

func (c *Contact) ContactRequestSent() {
	switch c.ContactRequestState {
	case ContactRequestStateNone, ContactRequestStateDismissed:
		c.ContactRequestState = ContactRequestStateSent
	case ContactRequestStateReceived:
		c.ContactRequestState = ContactRequestStateMutual
	}
}

func (c *Contact) ContactRequestReceived() {
	switch c.ContactRequestState {
	case ContactRequestStateNone:
		c.ContactRequestState = ContactRequestStateReceived
	case ContactRequestStateSent:
		c.ContactRequestState = ContactRequestStateMutual
	}
}

func (c *Contact) ContactRequestAccepted() {
	switch c.ContactRequestState {
	case ContactRequestStateSent:
		c.ContactRequestState = ContactRequestStateMutual
	}
}

func (c *Contact) AcceptContactRequest() {
	switch c.ContactRequestState {
	case ContactRequestStateReceived, ContactRequestStateDismissed:
		c.ContactRequestState = ContactRequestStateMutual
	}
}

func (c *Contact) RetractContactRequest() {
	c.ContactRequestState = ContactRequestStateNone
}

func (c *Contact) ContactRequestRetracted() {
	c.ContactRequestState = ContactRequestStateNone
}

func (c *Contact) DismissContactRequest() {
	c.ContactRequestState = ContactRequestStateDismissed
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
	id := common.PubkeyToHex(publicKey)
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

func contactIDFromPublicKeyString(key string) (string, error) {
	pubKey, err := common.HexToPubkey(key)
	if err != nil {
		return "", err
	}

	return contactIDFromPublicKey(pubKey), nil
}

func (c *Contact) MarshalJSON() ([]byte, error) {
	type Alias Contact
	item := struct {
		*Alias
		CompressedKey string `json:"compressedKey"`
	}{
		Alias: (*Alias)(c),
	}

	compressedKey, err := multiformat.SerializeLegacyKey(item.ID)
	if err != nil {
		return nil, err
	}
	item.CompressedKey = compressedKey

	return json.Marshal(item)
}
