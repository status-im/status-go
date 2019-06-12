package client

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

//go:generate stringer -type=ContactType

// ContactType defines a type of a contact.
type ContactType int

func (c ContactType) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, c)), nil
}

func (c *ContactType) UnmarshalJSON(data []byte) error {
	switch string(data) {
	case fmt.Sprintf(`"%s"`, ContactPublicRoom):
		*c = ContactPublicRoom
	case fmt.Sprintf(`"%s"`, ContactPrivate):
		*c = ContactPrivate
	default:
		return fmt.Errorf("invalid ContactType: %s", data)
	}

	return nil
}

// Types of contacts.
const (
	ContactPublicRoom ContactType = iota + 1
	ContactPrivate
)

// ContactState defines state of the contact.
type ContactState int

const (
	// ContactAdded default level. Added or confirmed by user.
	ContactAdded ContactState = iota + 1
	// ContactNew contact got connected to us and waits for being added or blocked.
	ContactNew
	// ContactBlocked means that all incoming messages from it will be discarded.
	ContactBlocked
)

// Contact is a single contact which has a type and name.
type Contact struct {
	Name      string           `json:"name"`
	Type      ContactType      `json:"type"`
	State     ContactState     `json:"state"`
	Topic     string           `json:"topic"`
	PublicKey *ecdsa.PublicKey `json:"-"`
}

// CreateContactPrivate creates a new private contact.
func CreateContactPrivate(name, pubKeyHex string, state ContactState) (c Contact, err error) {
	pubKeyBytes, err := hexutil.Decode(pubKeyHex)
	if err != nil {
		return
	}

	c.Name = name
	c.Type = ContactPrivate
	c.State = state
	c.Topic = DefaultPrivateTopic()
	c.PublicKey, err = crypto.UnmarshalPubkey(pubKeyBytes)

	return
}

// CreateContactPublicRoom creates a public room contact.
func CreateContactPublicRoom(name string, state ContactState) Contact {
	return Contact{
		Name:  name,
		Type:  ContactPublicRoom,
		State: state,
		Topic: name,
	}
}

// String returns a string representation of Contact.
func (c Contact) String() string {
	return c.Name
}

// Equal returns true if contacts have same name and same type.
func (c Contact) Equal(other Contact) bool {
	return c.Name == other.Name && c.Type == other.Type
}

func (c Contact) MarshalJSON() ([]byte, error) {
	type ContactAlias Contact

	item := struct {
		ContactAlias
		PublicKey string `json:"public_key,omitempty"`
	}{
		ContactAlias: ContactAlias(c),
	}

	if c.PublicKey != nil {
		item.PublicKey = EncodePublicKeyAsString(c.PublicKey)
	}

	return json.Marshal(&item)
}

func (c *Contact) UnmarshalJSON(data []byte) error {
	type ContactAlias Contact

	var item struct {
		*ContactAlias
		PublicKey string `json:"public_key,omitempty"`
	}

	if err := json.Unmarshal(data, &item); err != nil {
		return err
	}

	if len(item.PublicKey) > 2 {
		pubKey, err := hexutil.Decode(item.PublicKey)
		if err != nil {
			return err
		}

		item.ContactAlias.PublicKey, err = crypto.UnmarshalPubkey(pubKey)
		if err != nil {
			return err
		}
	}

	*c = *(*Contact)(item.ContactAlias)

	return nil
}

// EncodePublicKeyAsString encodes a public key as a string.
// It starts with 0x to indicate it's hex encoding.
func EncodePublicKeyAsString(pubKey *ecdsa.PublicKey) string {
	return hexutil.Encode(crypto.FromECDSAPub(pubKey))
}
