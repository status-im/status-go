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
	if profilePicturesVisibility == settings.ProfilePicturesVisibilityNone || (profilePicturesVisibility == settings.ProfilePicturesVisibilityContactsOnly && !c.added()) {
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

	Blocked bool `json:"blocked"`

	// ContactRequestRemoteState is the state of the contact request
	// on the contact's end
	ContactRequestRemoteState ContactRequestState `json:"contactRequestRemoteState"`
	// ContactRequestRemoteClock is the clock for incoming contact requests
	ContactRequestRemoteClock uint64 `json:"contactRequestRemoteClock"`

	// ContactRequestLocalState is the state of the contact request
	// on our end
	ContactRequestLocalState ContactRequestState `json:"contactRequestLocalState"`
	// ContactRequestLocalClock is the clock for outgoing contact requests
	ContactRequestLocalClock uint64 `json:"contactRequestLocalClock"`

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

func (c *Contact) Block(clock uint64) {
	c.Blocked = true
	c.DismissContactRequest(clock)
}

func (c *Contact) BlockDesktop() {
	c.Blocked = true
}

func (c *Contact) Unblock(clock uint64) {
	c.Blocked = false
	// Reset the contact request flow
	c.RetractContactRequest(clock)
}

func (c *Contact) added() bool {
	return c.ContactRequestLocalState == ContactRequestStateSent
}

func (c *Contact) hasAddedUs() bool {
	return c.ContactRequestRemoteState == ContactRequestStateReceived
}

func (c *Contact) mutual() bool {
	return c.added() && c.hasAddedUs()
}

type ContactRequestProcessingResponse struct {
	processed                 bool
	newContactRequestReceived bool
}

func (c *Contact) ContactRequestSent(clock uint64) ContactRequestProcessingResponse {
	if clock <= c.ContactRequestLocalClock {
		return ContactRequestProcessingResponse{}
	}

	c.ContactRequestLocalClock = clock
	c.ContactRequestLocalState = ContactRequestStateSent

	c.Removed = false

	return ContactRequestProcessingResponse{processed: true}
}

func (c *Contact) AcceptContactRequest(clock uint64) ContactRequestProcessingResponse {
	// We treat accept the same as sent, that's because accepting a contact
	// request that does not exist is possible if the instruction is coming from
	// a different device, we'd rather assume that a contact requested existed
	// and didn't reach our device than being in an inconsistent state
	return c.ContactRequestSent(clock)
}

func (c *Contact) RetractContactRequest(clock uint64) ContactRequestProcessingResponse {
	if clock <= c.ContactRequestLocalClock {
		return ContactRequestProcessingResponse{}
	}

	// This is a symmetric action, we set both local & remote clock
	// since we want everything before this point discarded, regardless
	// the side it was sent from
	c.ContactRequestLocalClock = clock
	c.ContactRequestLocalState = ContactRequestStateNone
	c.ContactRequestRemoteState = ContactRequestStateNone
	c.ContactRequestRemoteClock = clock
	c.Removed = true

	return ContactRequestProcessingResponse{processed: true}
}

func (c *Contact) DismissContactRequest(clock uint64) ContactRequestProcessingResponse {
	if clock <= c.ContactRequestLocalClock {
		return ContactRequestProcessingResponse{}
	}

	c.ContactRequestLocalClock = clock
	c.ContactRequestLocalState = ContactRequestStateDismissed

	return ContactRequestProcessingResponse{processed: true}
}

// Remote actions

func (c *Contact) ContactRequestRetracted(clock uint64) ContactRequestProcessingResponse {
	if clock <= c.ContactRequestRemoteClock {
		return ContactRequestProcessingResponse{}
	}

	// This is a symmetric action, we set both local & remote clock
	// since we want everything before this point discarded, regardless
	// the side it was sent from
	c.ContactRequestRemoteClock = clock
	c.ContactRequestRemoteState = ContactRequestStateNone
	c.ContactRequestLocalClock = clock
	c.ContactRequestLocalState = ContactRequestStateNone

	return ContactRequestProcessingResponse{processed: true}
}

func (c *Contact) ContactRequestReceived(clock uint64) ContactRequestProcessingResponse {
	if clock <= c.ContactRequestRemoteClock {
		return ContactRequestProcessingResponse{}
	}

	r := ContactRequestProcessingResponse{processed: true}
	c.ContactRequestRemoteClock = clock
	switch c.ContactRequestRemoteState {
	case ContactRequestStateNone:
		r.newContactRequestReceived = true
	}
	c.ContactRequestRemoteState = ContactRequestStateReceived

	return r
}

func (c *Contact) ContactRequestAccepted(clock uint64) ContactRequestProcessingResponse {
	if clock <= c.ContactRequestRemoteClock {
		return ContactRequestProcessingResponse{}
	}
	// We treat received and accepted in the same way
	// since the intention is clear on the other side
	// and there's no difference
	return c.ContactRequestReceived(clock)
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

func (c *Contact) processSyncContactRequestState(remoteState ContactRequestState, remoteClock uint64, localState ContactRequestState, localClock uint64) {
	// We process the two separately, first local state
	switch localState {
	case ContactRequestStateDismissed:
		c.DismissContactRequest(localClock)
	case ContactRequestStateNone:
		c.RetractContactRequest(localClock)
	case ContactRequestStateSent:
		c.ContactRequestSent(localClock)
	}

	// and later remote state
	switch remoteState {
	case ContactRequestStateReceived:
		c.ContactRequestReceived(remoteClock)
	case ContactRequestStateNone:
		c.ContactRequestRetracted(remoteClock)
	}
}

func (c *Contact) MarshalJSON() ([]byte, error) {
	type Alias Contact
	item := struct {
		*Alias
		CompressedKey       string              `json:"compressedKey"`
		Added               bool                `json:"added"`
		ContactRequestState ContactRequestState `json:"contactRequestState"`
		HasAddedUs          bool                `json:"hasAddedUs"`
	}{
		Alias: (*Alias)(c),
	}

	compressedKey, err := multiformat.SerializeLegacyKey(item.ID)
	if err != nil {
		return nil, err
	}
	item.CompressedKey = compressedKey

	item.Added = c.added()
	item.HasAddedUs = c.hasAddedUs()

	if c.mutual() {
		item.ContactRequestState = ContactRequestStateMutual
	} else if c.added() {
		item.ContactRequestState = ContactRequestStateSent
	} else if c.hasAddedUs() {
		item.ContactRequestState = ContactRequestStateReceived
	}

	return json.Marshal(item)
}
