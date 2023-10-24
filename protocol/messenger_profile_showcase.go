package protocol

import (
	"crypto/ecdsa"
	crand "crypto/rand"
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/identity"
	"github.com/status-im/status-go/protocol/protobuf"
)

type ProfileShowcasePreferences struct {
	Communities  []*ProfileShowcaseEntry `json:"communities"`
	Accounts     []*ProfileShowcaseEntry `json:"accounts"`
	Collectibles []*ProfileShowcaseEntry `json:"collectibles"`
	Assets       []*ProfileShowcaseEntry `json:"assets"`
}

func toProfileShowcaseUpdateEntries(entries []*ProfileShowcaseEntry, visibility ProfileShowcaseVisibility) []*protobuf.ProfileShowcaseEntry {
	result := []*protobuf.ProfileShowcaseEntry{}

	for _, entry := range entries {
		if entry.ShowcaseVisibility != visibility {
			continue
		}
		update := &protobuf.ProfileShowcaseEntry{
			EntryId: entry.ID,
			Order:   uint32(entry.Order),
		}
		result = append(result, update)
	}
	return result
}

func fromProfileShowcaseUpdateEntries(messages []*protobuf.ProfileShowcaseEntry) []*identity.VisibleProfileShowcaseEntry {
	entries := []*identity.VisibleProfileShowcaseEntry{}
	for _, entry := range messages {
		entries = append(entries, &identity.VisibleProfileShowcaseEntry{
			EntryID: entry.EntryId,
			Order:   int(entry.Order),
		})
	}
	return entries
}

func (p *ProfileShowcasePreferences) Validate() error {
	for _, community := range p.Communities {
		if community.EntryType != ProfileShowcaseEntryTypeCommunity {
			return fmt.Errorf("communities must contain only entriers of type ProfileShowcaseEntryTypeCommunity")
		}
	}

	for _, community := range p.Accounts {
		if community.EntryType != ProfileShowcaseEntryTypeAccount {
			return fmt.Errorf("accounts must contain only entriers of type ProfileShowcaseEntryTypeAccount")
		}
	}

	for _, community := range p.Collectibles {
		if community.EntryType != ProfileShowcaseEntryTypeCollectible {
			return fmt.Errorf("collectibles must contain only entriers of type ProfileShowcaseEntryTypeCollectible")
		}
	}

	for _, community := range p.Assets {
		if community.EntryType != ProfileShowcaseEntryTypeAsset {
			return fmt.Errorf("assets must contain only entriers of type ProfileShowcaseEntryTypeAsset")
		}
	}
	return nil
}

func (m *Messenger) SetProfileShowcasePreferences(preferences ProfileShowcasePreferences) error {
	err := preferences.Validate()
	if err != nil {
		return err
	}

	allPreferences := []*ProfileShowcaseEntry{}

	allPreferences = append(allPreferences, preferences.Communities...)
	allPreferences = append(allPreferences, preferences.Accounts...)
	allPreferences = append(allPreferences, preferences.Collectibles...)
	allPreferences = append(allPreferences, preferences.Assets...)

	err = m.persistence.SaveProfileShowcasePreferences(allPreferences)
	if err != nil {
		return err
	}

	return m.publishContactCode()
}

func (m *Messenger) GetProfileShowcasePreferences() (*ProfileShowcasePreferences, error) {
	// NOTE: in the future default profile preferences should be filled in for each group according to special rules,
	// that's why they should be grouped here
	communities, err := m.persistence.GetProfileShowcasePreferencesByType(ProfileShowcaseEntryTypeCommunity)
	if err != nil {
		return nil, err
	}

	accounts, err := m.persistence.GetProfileShowcasePreferencesByType(ProfileShowcaseEntryTypeAccount)
	if err != nil {
		return nil, err
	}

	collectibles, err := m.persistence.GetProfileShowcasePreferencesByType(ProfileShowcaseEntryTypeCollectible)
	if err != nil {
		return nil, err
	}

	assets, err := m.persistence.GetProfileShowcasePreferencesByType(ProfileShowcaseEntryTypeAsset)
	if err != nil {
		return nil, err
	}

	return &ProfileShowcasePreferences{
		Communities:  communities,
		Accounts:     accounts,
		Collectibles: collectibles,
		Assets:       assets,
	}, nil
}

func (m *Messenger) EncryptProfileShowcaseEntriesWithContactPubKeys(entries *protobuf.ProfileShowcaseEntries, contacts []*Contact) (*protobuf.ProfileShowcaseEntriesEncrypted, error) {
	// Make AES key
	AESKey := make([]byte, 32)
	_, err := crand.Read(AESKey)
	if err != nil {
		return nil, err
	}

	// Encrypt showcase entries with the AES key
	data, err := proto.Marshal(entries)
	if err != nil {
		return nil, err
	}

	encrypted, err := common.Encrypt(data, AESKey, crand.Reader)
	if err != nil {
		return nil, err
	}

	eAESKeys := [][]byte{}
	// Sign for each contact
	for _, contact := range contacts {
		var pubK *ecdsa.PublicKey
		var sharedKey []byte
		var eAESKey []byte

		pubK, err = contact.PublicKey()
		if err != nil {
			return nil, err
		}
		// Generate a Diffie-Helman (DH) between the sender private key and the recipient's public key
		sharedKey, err = common.MakeECDHSharedKey(m.identity, pubK)
		if err != nil {
			return nil, err
		}

		// Encrypt the main AES key with AES encryption using the DH key
		eAESKey, err = common.Encrypt(AESKey, sharedKey, crand.Reader)
		if err != nil {
			return nil, err
		}

		eAESKeys = append(eAESKeys, eAESKey)
	}

	return &protobuf.ProfileShowcaseEntriesEncrypted{
		EncryptedEntries: encrypted,
		EncryptionKeys:   eAESKeys,
	}, nil
}

func (m *Messenger) DecryptProfileShowcaseEntriesWithContactPubKeys(senderPubKey *ecdsa.PublicKey, encrypted *protobuf.ProfileShowcaseEntriesEncrypted) (*protobuf.ProfileShowcaseEntries, error) {
	for _, eAESKey := range encrypted.EncryptionKeys {
		// Generate a Diffie-Helman (DH) between the recipient's private key and the sender's public key
		sharedKey, err := common.MakeECDHSharedKey(m.identity, senderPubKey)
		if err != nil {
			return nil, err
		}

		// Decrypt the main encryption AES key with AES encryption using the DH key
		dAESKey, err := common.Decrypt(eAESKey, sharedKey)
		if err != nil {
			if err.Error() == CipherMessageAutentificationFailed {
				continue
			}
			return nil, err
		}
		if dAESKey == nil {
			return nil, errors.New("decrypting the payload encryption key resulted in no error and a nil key")
		}

		// Decrypt profile entries with the newly decrypted main encryption AES key
		entriesData, err := common.Decrypt(encrypted.EncryptedEntries, dAESKey)
		if err != nil {
			return nil, err
		}

		entries := &protobuf.ProfileShowcaseEntries{}
		err = proto.Unmarshal(entriesData, entries)
		if err != nil {
			return nil, err
		}

		return entries, nil
	}

	// Return empty if no matching key found
	return &protobuf.ProfileShowcaseEntries{}, nil
}

func (m *Messenger) GetProfileShowcaseForSelfIdentity() (*protobuf.ProfileShowcase, error) {
	preferences, err := m.GetProfileShowcasePreferences()
	if err != nil {
		return nil, err
	}

	forEveryone := &protobuf.ProfileShowcaseEntries{
		Communities:  toProfileShowcaseUpdateEntries(preferences.Communities, ProfileShowcaseVisibilityEveryone),
		Accounts:     toProfileShowcaseUpdateEntries(preferences.Accounts, ProfileShowcaseVisibilityEveryone),
		Collectibles: toProfileShowcaseUpdateEntries(preferences.Collectibles, ProfileShowcaseVisibilityEveryone),
		Assets:       toProfileShowcaseUpdateEntries(preferences.Assets, ProfileShowcaseVisibilityEveryone),
	}

	forContacts := &protobuf.ProfileShowcaseEntries{
		Communities:  toProfileShowcaseUpdateEntries(preferences.Communities, ProfileShowcaseVisibilityContacts),
		Accounts:     toProfileShowcaseUpdateEntries(preferences.Accounts, ProfileShowcaseVisibilityContacts),
		Collectibles: toProfileShowcaseUpdateEntries(preferences.Collectibles, ProfileShowcaseVisibilityContacts),
		Assets:       toProfileShowcaseUpdateEntries(preferences.Assets, ProfileShowcaseVisibilityContacts),
	}

	forIDVerifiedContacts := &protobuf.ProfileShowcaseEntries{
		Communities:  toProfileShowcaseUpdateEntries(preferences.Communities, ProfileShowcaseVisibilityIDVerifiedContacts),
		Accounts:     toProfileShowcaseUpdateEntries(preferences.Accounts, ProfileShowcaseVisibilityIDVerifiedContacts),
		Collectibles: toProfileShowcaseUpdateEntries(preferences.Collectibles, ProfileShowcaseVisibilityIDVerifiedContacts),
		Assets:       toProfileShowcaseUpdateEntries(preferences.Assets, ProfileShowcaseVisibilityIDVerifiedContacts),
	}

	mutualContacts := []*Contact{}
	iDVerifiedContacts := []*Contact{}

	m.allContacts.Range(func(_ string, contact *Contact) (shouldContinue bool) {
		if contact.mutual() {
			mutualContacts = append(mutualContacts, contact)
			if contact.IsVerified() {
				iDVerifiedContacts = append(iDVerifiedContacts, contact)
			}
		}
		return true
	})

	forContactsEncrypted, err := m.EncryptProfileShowcaseEntriesWithContactPubKeys(forContacts, mutualContacts)
	if err != nil {
		return nil, err
	}

	forIDVerifiedContactsEncrypted, err := m.EncryptProfileShowcaseEntriesWithContactPubKeys(forIDVerifiedContacts, iDVerifiedContacts)
	if err != nil {
		return nil, err
	}

	return &protobuf.ProfileShowcase{
		ForEveryone:           forEveryone,
		ForContacts:           forContactsEncrypted,
		ForIdVerifiedContacts: forIDVerifiedContactsEncrypted,
	}, nil
}

func (m *Messenger) BuildProfileShowcaseFromIdentity(senderPubKey *ecdsa.PublicKey, message *protobuf.ProfileShowcase) (*identity.ProfileShowcase, error) {
	communities := []*identity.VisibleProfileShowcaseEntry{}
	accounts := []*identity.VisibleProfileShowcaseEntry{}
	collectibles := []*identity.VisibleProfileShowcaseEntry{}
	assets := []*identity.VisibleProfileShowcaseEntry{}

	communities = append(communities, fromProfileShowcaseUpdateEntries(message.ForEveryone.Communities)...)
	accounts = append(accounts, fromProfileShowcaseUpdateEntries(message.ForEveryone.Accounts)...)
	collectibles = append(collectibles, fromProfileShowcaseUpdateEntries(message.ForEveryone.Collectibles)...)
	assets = append(assets, fromProfileShowcaseUpdateEntries(message.ForEveryone.Assets)...)

	forContacts, err := m.DecryptProfileShowcaseEntriesWithContactPubKeys(senderPubKey, message.ForContacts)
	if err != nil {
		return nil, err
	}

	if forContacts != nil {
		communities = append(communities, fromProfileShowcaseUpdateEntries(forContacts.Communities)...)
		accounts = append(accounts, fromProfileShowcaseUpdateEntries(forContacts.Accounts)...)
		collectibles = append(collectibles, fromProfileShowcaseUpdateEntries(forContacts.Collectibles)...)
		assets = append(assets, fromProfileShowcaseUpdateEntries(forContacts.Assets)...)
	}

	forIDVerifiedContacts, err := m.DecryptProfileShowcaseEntriesWithContactPubKeys(senderPubKey, message.ForIdVerifiedContacts)
	if err != nil {
		return nil, err
	}

	if forIDVerifiedContacts != nil {
		communities = append(communities, fromProfileShowcaseUpdateEntries(forIDVerifiedContacts.Communities)...)
		accounts = append(accounts, fromProfileShowcaseUpdateEntries(forIDVerifiedContacts.Accounts)...)
		collectibles = append(collectibles, fromProfileShowcaseUpdateEntries(forIDVerifiedContacts.Collectibles)...)
		assets = append(assets, fromProfileShowcaseUpdateEntries(forIDVerifiedContacts.Assets)...)
	}

	return &identity.ProfileShowcase{
		Communities:  communities,
		Accounts:     accounts,
		Collectibles: collectibles,
		Assets:       assets,
	}, nil
}
