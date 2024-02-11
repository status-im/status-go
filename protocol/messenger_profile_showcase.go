package protocol

import (
	"crypto/ecdsa"
	crand "crypto/rand"
	"errors"
	"reflect"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/identity"
	"github.com/status-im/status-go/protocol/protobuf"
)

var errorNoAccountProvidedWithTokenOrCollectible = errors.New("no account provided with tokens or collectible")
var errorDublicateAccountAddress = errors.New("duplicate account address")

// NOTE: this error is temporary unused because we don't know account on this stage
// var errorNoAccountAddressForCollectible = errors.New("no account found for collectible")
var errorAccountVisibilityLowerThanCollectible = errors.New("account visibility lower than collectible")
var errorDecryptingPayloadEncryptionKey = errors.New("decrypting the payload encryption key resulted in no error and a nil key")

func toProfileShowcaseCommunityProto(preferences []*identity.ProfileShowcaseCommunityPreference, visibility identity.ProfileShowcaseVisibility) []*protobuf.ProfileShowcaseCommunity {
	communities := []*protobuf.ProfileShowcaseCommunity{}
	for _, preference := range preferences {
		if preference.ShowcaseVisibility != visibility {
			continue
		}

		communities = append(communities, &protobuf.ProfileShowcaseCommunity{
			CommunityId: preference.CommunityID,
			Order:       uint32(preference.Order),
		})
	}
	return communities
}

func toProfileShowcaseAccountProto(preferences []*identity.ProfileShowcaseAccountPreference, visibility identity.ProfileShowcaseVisibility) []*protobuf.ProfileShowcaseAccount {
	accounts := []*protobuf.ProfileShowcaseAccount{}
	for _, preference := range preferences {
		if preference.ShowcaseVisibility != visibility {
			continue
		}

		accounts = append(accounts, &protobuf.ProfileShowcaseAccount{
			Address: preference.Address,
			Name:    preference.Name,
			ColorId: preference.ColorID,
			Emoji:   preference.Emoji,
			Order:   uint32(preference.Order),
		})
	}
	return accounts
}

func toProfileShowcaseCollectibleProto(preferences []*identity.ProfileShowcaseCollectiblePreference, visibility identity.ProfileShowcaseVisibility) []*protobuf.ProfileShowcaseCollectible {
	collectibles := []*protobuf.ProfileShowcaseCollectible{}
	for _, preference := range preferences {
		if preference.ShowcaseVisibility != visibility {
			continue
		}

		collectibles = append(collectibles, &protobuf.ProfileShowcaseCollectible{
			ContractAddress: preference.ContractAddress,
			ChainId:         preference.ChainID,
			TokenId:         preference.TokenID,
			CommunityId:     preference.CommunityID,
			AccountAddress:  preference.AccountAddress,
			Order:           uint32(preference.Order),
		})
	}
	return collectibles
}

func toProfileShowcaseVerifiedTokensProto(preferences []*identity.ProfileShowcaseVerifiedTokenPreference, visibility identity.ProfileShowcaseVisibility) []*protobuf.ProfileShowcaseVerifiedToken {
	tokens := []*protobuf.ProfileShowcaseVerifiedToken{}
	for _, preference := range preferences {
		if preference.ShowcaseVisibility != visibility {
			continue
		}

		tokens = append(tokens, &protobuf.ProfileShowcaseVerifiedToken{
			Symbol: preference.Symbol,
			Order:  uint32(preference.Order),
		})
	}
	return tokens
}

func toProfileShowcaseUnverifiedTokensProto(preferences []*identity.ProfileShowcaseUnverifiedTokenPreference, visibility identity.ProfileShowcaseVisibility) []*protobuf.ProfileShowcaseUnverifiedToken {
	tokens := []*protobuf.ProfileShowcaseUnverifiedToken{}
	for _, preference := range preferences {
		if preference.ShowcaseVisibility != visibility {
			continue
		}

		tokens = append(tokens, &protobuf.ProfileShowcaseUnverifiedToken{
			ContractAddress: preference.ContractAddress,
			ChainId:         preference.ChainID,
			Order:           uint32(preference.Order),
		})
	}
	return tokens
}

func fromProfileShowcaseCommunityProto(messages []*protobuf.ProfileShowcaseCommunity) []*identity.ProfileShowcaseCommunity {
	communities := []*identity.ProfileShowcaseCommunity{}
	for _, entry := range messages {
		communities = append(communities, &identity.ProfileShowcaseCommunity{
			CommunityID: entry.CommunityId,
			Order:       int(entry.Order),
		})
	}
	return communities
}

func fromProfileShowcaseAccountProto(messages []*protobuf.ProfileShowcaseAccount) []*identity.ProfileShowcaseAccount {
	accounts := []*identity.ProfileShowcaseAccount{}
	for _, entry := range messages {
		accounts = append(accounts, &identity.ProfileShowcaseAccount{
			Address: entry.Address,
			Name:    entry.Name,
			ColorID: entry.ColorId,
			Emoji:   entry.Emoji,
			Order:   int(entry.Order),
		})
	}
	return accounts
}

func fromProfileShowcaseCollectibleProto(messages []*protobuf.ProfileShowcaseCollectible) []*identity.ProfileShowcaseCollectible {
	collectibles := []*identity.ProfileShowcaseCollectible{}
	for _, entry := range messages {
		collectibles = append(collectibles, &identity.ProfileShowcaseCollectible{
			ContractAddress: entry.ContractAddress,
			ChainID:         entry.ChainId,
			TokenID:         entry.TokenId,
			CommunityID:     entry.CommunityId,
			AccountAddress:  entry.AccountAddress,
			Order:           int(entry.Order),
		})
	}
	return collectibles
}

func fromProfileShowcaseVerifiedTokenProto(messages []*protobuf.ProfileShowcaseVerifiedToken) []*identity.ProfileShowcaseVerifiedToken {
	tokens := []*identity.ProfileShowcaseVerifiedToken{}
	for _, entry := range messages {
		tokens = append(tokens, &identity.ProfileShowcaseVerifiedToken{
			Symbol: entry.Symbol,
			Order:  int(entry.Order),
		})
	}
	return tokens
}

func fromProfileShowcaseUnverifiedTokenProto(messages []*protobuf.ProfileShowcaseUnverifiedToken) []*identity.ProfileShowcaseUnverifiedToken {
	tokens := []*identity.ProfileShowcaseUnverifiedToken{}
	for _, entry := range messages {
		tokens = append(tokens, &identity.ProfileShowcaseUnverifiedToken{
			ContractAddress: entry.ContractAddress,
			ChainID:         entry.ChainId,
			Order:           int(entry.Order),
		})
	}
	return tokens
}

func Validate(preferences *identity.ProfileShowcasePreferences) error {
	if (len(preferences.VerifiedTokens) > 0 || len(preferences.UnverifiedTokens) > 0 || len(preferences.Collectibles) > 0) &&
		len(preferences.Accounts) == 0 {
		return errorNoAccountProvidedWithTokenOrCollectible
	}

	accountsMap := make(map[string]*identity.ProfileShowcaseAccountPreference)
	for _, account := range preferences.Accounts {
		if _, ok := accountsMap[account.Address]; ok {
			return errorDublicateAccountAddress
		}
		accountsMap[account.Address] = account
	}

	for _, collectible := range preferences.Collectibles {
		account, ok := accountsMap[collectible.AccountAddress]
		if !ok {
			return nil
			// NOTE: with current wallet collectible implementation we don't know account on this stage
			// return errorNoAccountAddressForCollectible
		}
		if account.ShowcaseVisibility < collectible.ShowcaseVisibility {
			return errorAccountVisibilityLowerThanCollectible
		}
	}

	return nil
}

func (m *Messenger) SetProfileShowcasePreferences(preferences *identity.ProfileShowcasePreferences) error {
	err := Validate(preferences)
	if err != nil {
		return err
	}

	err = m.persistence.SaveProfileShowcasePreferences(preferences)
	if err != nil {
		return err
	}

	return m.DispatchProfileShowcase()
}

func (m *Messenger) DispatchProfileShowcase() error {
	return m.publishContactCode()
}

func (m *Messenger) GetProfileShowcasePreferences() (*identity.ProfileShowcasePreferences, error) {
	return m.persistence.GetProfileShowcasePreferences()
}

func (m *Messenger) GetProfileShowcaseForContact(contactID string) (*identity.ProfileShowcase, error) {
	return m.persistence.GetProfileShowcaseForContact(contactID)
}

func (m *Messenger) GetProfileShowcaseAccountsByAddress(address string) ([]*identity.ProfileShowcaseAccount, error) {
	return m.persistence.GetProfileShowcaseAccountsByAddress(address)
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

func (m *Messenger) DecryptProfileShowcaseEntriesWithPubKey(senderPubKey *ecdsa.PublicKey, encrypted *protobuf.ProfileShowcaseEntriesEncrypted) (*protobuf.ProfileShowcaseEntries, error) {
	for _, eAESKey := range encrypted.EncryptionKeys {
		// Generate a Diffie-Helman (DH) between the recipient's private key and the sender's public key
		sharedKey, err := common.MakeECDHSharedKey(m.identity, senderPubKey)
		if err != nil {
			return nil, err
		}

		// Decrypt the main encryption AES key with AES encryption using the DH key
		dAESKey, err := common.Decrypt(eAESKey, sharedKey)
		if err != nil {
			if err.Error() == ErrCipherMessageAutentificationFailed {
				continue
			}
			return nil, err
		}
		if dAESKey == nil {
			return nil, errorDecryptingPayloadEncryptionKey
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
		Communities:      toProfileShowcaseCommunityProto(preferences.Communities, identity.ProfileShowcaseVisibilityEveryone),
		Accounts:         toProfileShowcaseAccountProto(preferences.Accounts, identity.ProfileShowcaseVisibilityEveryone),
		Collectibles:     toProfileShowcaseCollectibleProto(preferences.Collectibles, identity.ProfileShowcaseVisibilityEveryone),
		VerifiedTokens:   toProfileShowcaseVerifiedTokensProto(preferences.VerifiedTokens, identity.ProfileShowcaseVisibilityEveryone),
		UnverifiedTokens: toProfileShowcaseUnverifiedTokensProto(preferences.UnverifiedTokens, identity.ProfileShowcaseVisibilityEveryone),
	}

	forContacts := &protobuf.ProfileShowcaseEntries{
		Communities:      toProfileShowcaseCommunityProto(preferences.Communities, identity.ProfileShowcaseVisibilityContacts),
		Accounts:         toProfileShowcaseAccountProto(preferences.Accounts, identity.ProfileShowcaseVisibilityContacts),
		Collectibles:     toProfileShowcaseCollectibleProto(preferences.Collectibles, identity.ProfileShowcaseVisibilityContacts),
		VerifiedTokens:   toProfileShowcaseVerifiedTokensProto(preferences.VerifiedTokens, identity.ProfileShowcaseVisibilityContacts),
		UnverifiedTokens: toProfileShowcaseUnverifiedTokensProto(preferences.UnverifiedTokens, identity.ProfileShowcaseVisibilityContacts),
	}

	forIDVerifiedContacts := &protobuf.ProfileShowcaseEntries{
		Communities:      toProfileShowcaseCommunityProto(preferences.Communities, identity.ProfileShowcaseVisibilityIDVerifiedContacts),
		Accounts:         toProfileShowcaseAccountProto(preferences.Accounts, identity.ProfileShowcaseVisibilityIDVerifiedContacts),
		Collectibles:     toProfileShowcaseCollectibleProto(preferences.Collectibles, identity.ProfileShowcaseVisibilityIDVerifiedContacts),
		VerifiedTokens:   toProfileShowcaseVerifiedTokensProto(preferences.VerifiedTokens, identity.ProfileShowcaseVisibilityIDVerifiedContacts),
		UnverifiedTokens: toProfileShowcaseUnverifiedTokensProto(preferences.UnverifiedTokens, identity.ProfileShowcaseVisibilityIDVerifiedContacts),
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

func (m *Messenger) BuildProfileShowcaseFromIdentity(state *ReceivedMessageState, message *protobuf.ProfileShowcase) error {
	communities := []*identity.ProfileShowcaseCommunity{}
	accounts := []*identity.ProfileShowcaseAccount{}
	collectibles := []*identity.ProfileShowcaseCollectible{}
	verifiedTokens := []*identity.ProfileShowcaseVerifiedToken{}
	unverifiedTokens := []*identity.ProfileShowcaseUnverifiedToken{}

	communities = append(communities, fromProfileShowcaseCommunityProto(message.ForEveryone.Communities)...)
	accounts = append(accounts, fromProfileShowcaseAccountProto(message.ForEveryone.Accounts)...)
	collectibles = append(collectibles, fromProfileShowcaseCollectibleProto(message.ForEveryone.Collectibles)...)
	verifiedTokens = append(verifiedTokens, fromProfileShowcaseVerifiedTokenProto(message.ForEveryone.VerifiedTokens)...)
	unverifiedTokens = append(unverifiedTokens, fromProfileShowcaseUnverifiedTokenProto(message.ForEveryone.UnverifiedTokens)...)

	senderPubKey := state.CurrentMessageState.PublicKey
	contactID := state.CurrentMessageState.Contact.ID

	forContacts, err := m.DecryptProfileShowcaseEntriesWithPubKey(senderPubKey, message.ForContacts)
	if err != nil {
		return err
	}

	if forContacts != nil {
		communities = append(communities, fromProfileShowcaseCommunityProto(forContacts.Communities)...)
		accounts = append(accounts, fromProfileShowcaseAccountProto(forContacts.Accounts)...)
		collectibles = append(collectibles, fromProfileShowcaseCollectibleProto(forContacts.Collectibles)...)
		verifiedTokens = append(verifiedTokens, fromProfileShowcaseVerifiedTokenProto(forContacts.VerifiedTokens)...)
		unverifiedTokens = append(unverifiedTokens, fromProfileShowcaseUnverifiedTokenProto(forContacts.UnverifiedTokens)...)
	}

	forIDVerifiedContacts, err := m.DecryptProfileShowcaseEntriesWithPubKey(senderPubKey, message.ForIdVerifiedContacts)
	if err != nil {
		return err
	}

	if forIDVerifiedContacts != nil {
		communities = append(communities, fromProfileShowcaseCommunityProto(forIDVerifiedContacts.Communities)...)
		accounts = append(accounts, fromProfileShowcaseAccountProto(forIDVerifiedContacts.Accounts)...)
		collectibles = append(collectibles, fromProfileShowcaseCollectibleProto(forIDVerifiedContacts.Collectibles)...)
		verifiedTokens = append(verifiedTokens, fromProfileShowcaseVerifiedTokenProto(forIDVerifiedContacts.VerifiedTokens)...)
		unverifiedTokens = append(unverifiedTokens, fromProfileShowcaseUnverifiedTokenProto(forIDVerifiedContacts.UnverifiedTokens)...)
	}

	// TODO: validate community membership here (https://github.com/status-im/status-desktop/issues/13081)
	// TODO: validate collectible ownership here (https://github.com/status-im/status-desktop/issues/13073)

	newShowcase := &identity.ProfileShowcase{
		ContactID:        contactID,
		Communities:      communities,
		Accounts:         accounts,
		Collectibles:     collectibles,
		VerifiedTokens:   verifiedTokens,
		UnverifiedTokens: unverifiedTokens,
	}

	oldShowcase, err := m.persistence.GetProfileShowcaseForContact(contactID)
	if err != nil {
		return err
	}

	if reflect.DeepEqual(newShowcase, oldShowcase) {
		return nil
	}

	err = m.persistence.ClearProfileShowcaseForContact(contactID)
	if err != nil {
		return err
	}

	err = m.persistence.SaveProfileShowcaseForContact(newShowcase)
	if err != nil {
		return err
	}

	state.Response.AddProfileShowcase(newShowcase)
	return nil
}

func (m *Messenger) UpdateProfileShowcaseWalletAccount(account *accounts.Account) error {
	profileAccount, err := m.persistence.GetProfileShowcaseAccountPreference(account.Address.Hex())
	if err != nil {
		return err
	}

	if profileAccount == nil {
		// No corresponding profile entry, exit
		return nil
	}

	profileAccount.Name = account.Name
	profileAccount.ColorID = string(account.ColorID)
	profileAccount.Emoji = account.Emoji

	err = m.persistence.SaveProfileShowcaseAccountPreference(profileAccount)
	if err != nil {
		return err
	}

	return m.DispatchProfileShowcase()
}

func (m *Messenger) DeleteProfileShowcaseWalletAccount(account *accounts.Account) error {
	deleted, err := m.persistence.DeleteProfileShowcaseAccountPreference(account.Address.Hex())
	if err != nil {
		return err
	}

	if deleted {
		return m.DispatchProfileShowcase()
	}
	return nil
}

func (m *Messenger) DeleteProfileShowcaseCommunity(community *communities.Community) error {
	deleted, err := m.persistence.DeleteProfileShowcaseCommunityPreference(community.IDString())
	if err != nil {
		return err
	}

	if deleted {
		return m.DispatchProfileShowcase()
	}
	return nil
}
