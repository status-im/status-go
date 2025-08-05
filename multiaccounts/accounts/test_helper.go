package accounts

import (
	accsmanagementcommon "github.com/status-im/status-go/accounts-management/common"
	"github.com/status-im/status-go/accounts-management/generator"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/common"
)

const (
	testProfileMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon"
	testSeedMnemonic    = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	testSeed2Mnemonic   = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about abandon"
)

func GetWatchOnlyAccountsForTest() []*Account {
	wo1 := &Account{
		Address: types.Address{0x11},
		Type:    AccountTypeWatch,
		Name:    "WatchOnlyAcc1",
		ColorID: common.CustomizationColorPrimary,
		Emoji:   "emoji-1",
	}
	wo2 := &Account{
		Address: types.Address{0x12},
		Type:    AccountTypeWatch,
		Name:    "WatchOnlyAcc2",
		ColorID: common.CustomizationColorPrimary,
		Emoji:   "emoji-1",
	}
	wo3 := &Account{
		Address: types.Address{0x13},
		Type:    AccountTypeWatch,
		Name:    "WatchOnlyAcc3",
		ColorID: common.CustomizationColorPrimary,
		Emoji:   "emoji-1",
	}

	return []*Account{wo1, wo2, wo3}
}

func GetProfileKeypairForTest(includeChatAccount bool, includeDefaultWalletAccount bool, includeAdditionalAccounts bool) (
	keypair *Keypair, acc *generator.Account, derivedAccs map[string]*generator.Account, err error) {
	acc, derivedAccs, err = generator.CreateAndDeriveAccountsFromMnemonic(testProfileMnemonic, []string{
		accsmanagementcommon.PathEIP1581Chat,
		accsmanagementcommon.PathDefaultWalletAccount,
		accsmanagementcommon.CustomWalletPath1,
		accsmanagementcommon.CustomWalletPath2,
	}, "")
	if err != nil {
		return
	}

	keypair = &Keypair{
		KeyUID:      acc.KeyUID(),
		Name:        "Profile Name",
		Type:        KeypairTypeProfile,
		DerivedFrom: acc.Address().Hex(),
	}

	if includeChatAccount {
		profileAccount := &Account{
			Address:               derivedAccs[accsmanagementcommon.PathEIP1581Chat].Address(),
			KeyUID:                keypair.KeyUID,
			Wallet:                false,
			Chat:                  true,
			Type:                  AccountTypeGenerated,
			Path:                  accsmanagementcommon.PathEIP1581Chat,
			PublicKey:             types.Hex2Bytes(derivedAccs[accsmanagementcommon.PathEIP1581Chat].PublicKeyHex()),
			Name:                  "Profile Name",
			Operable:              AccountFullyOperable,
			ProdPreferredChainIDs: "1",
			TestPreferredChainIDs: "5",
		}
		keypair.Accounts = append(keypair.Accounts, profileAccount)
	}

	if includeDefaultWalletAccount {
		defaultWalletAccount := &Account{
			Address:               derivedAccs[accsmanagementcommon.PathDefaultWalletAccount].Address(),
			KeyUID:                keypair.KeyUID,
			Wallet:                true,
			Chat:                  false,
			Type:                  AccountTypeGenerated,
			Path:                  accsmanagementcommon.PathDefaultWalletAccount,
			PublicKey:             types.Hex2Bytes(derivedAccs[accsmanagementcommon.PathDefaultWalletAccount].PublicKeyHex()),
			Name:                  "Generated Acc 1",
			Emoji:                 "emoji-1",
			ColorID:               common.CustomizationColorPrimary,
			Hidden:                false,
			Clock:                 0,
			Removed:               false,
			Operable:              AccountFullyOperable,
			ProdPreferredChainIDs: "1",
			TestPreferredChainIDs: "5",
		}
		keypair.Accounts = append(keypair.Accounts, defaultWalletAccount)
		keypair.LastUsedDerivationIndex = 0
	}

	if includeAdditionalAccounts {
		generatedWalletAccount1 := &Account{
			Address:               derivedAccs[accsmanagementcommon.CustomWalletPath1].Address(),
			KeyUID:                keypair.KeyUID,
			Wallet:                false,
			Chat:                  false,
			Type:                  AccountTypeGenerated,
			Path:                  accsmanagementcommon.CustomWalletPath1,
			PublicKey:             types.Hex2Bytes(derivedAccs[accsmanagementcommon.CustomWalletPath1].PublicKeyHex()),
			Name:                  "Generated Acc 2",
			Emoji:                 "emoji-2",
			ColorID:               common.CustomizationColorPrimary,
			Hidden:                false,
			Clock:                 0,
			Removed:               false,
			Operable:              AccountFullyOperable,
			ProdPreferredChainIDs: "1",
			TestPreferredChainIDs: "5",
		}
		keypair.Accounts = append(keypair.Accounts, generatedWalletAccount1)
		keypair.LastUsedDerivationIndex = 1

		generatedWalletAccount2 := &Account{
			Address:               derivedAccs[accsmanagementcommon.CustomWalletPath2].Address(),
			KeyUID:                keypair.KeyUID,
			Wallet:                false,
			Chat:                  false,
			Type:                  AccountTypeGenerated,
			Path:                  accsmanagementcommon.CustomWalletPath2,
			PublicKey:             types.Hex2Bytes(derivedAccs[accsmanagementcommon.CustomWalletPath2].PublicKeyHex()),
			Name:                  "Generated Acc 3",
			Emoji:                 "emoji-3",
			ColorID:               common.CustomizationColorPrimary,
			Hidden:                false,
			Clock:                 0,
			Removed:               false,
			Operable:              AccountFullyOperable,
			ProdPreferredChainIDs: "1",
			TestPreferredChainIDs: "5",
		}
		keypair.Accounts = append(keypair.Accounts, generatedWalletAccount2)
		keypair.LastUsedDerivationIndex = 2
	}

	return
}

func GetSeedImportedKeypair1ForTest() (keypair *Keypair, acc *generator.Account, derivedAccs map[string]*generator.Account, err error) {
	acc, derivedAccs, err = generator.CreateAndDeriveAccountsFromMnemonic(testSeedMnemonic, []string{
		accsmanagementcommon.PathDefaultWalletAccount,
		accsmanagementcommon.CustomWalletPath1,
	}, "")
	if err != nil {
		return
	}

	keypair = &Keypair{
		KeyUID:      acc.KeyUID(),
		Name:        "Seed Imported 1",
		Type:        KeypairTypeSeed,
		DerivedFrom: acc.Address().Hex(),
	}

	seedGeneratedWalletAccount1 := &Account{
		Address:   derivedAccs[accsmanagementcommon.PathDefaultWalletAccount].Address(),
		KeyUID:    keypair.KeyUID,
		Wallet:    false,
		Chat:      false,
		Type:      AccountTypeSeed,
		Path:      accsmanagementcommon.PathDefaultWalletAccount,
		PublicKey: types.Hex2Bytes(derivedAccs[accsmanagementcommon.PathDefaultWalletAccount].PublicKeyHex()),
		Name:      "Seed Impo 1 Acc 1",
		Emoji:     "emoji-1",
		ColorID:   common.CustomizationColorPrimary,
		Hidden:    false,
		Clock:     0,
		Removed:   false,
		Operable:  AccountFullyOperable,
	}
	keypair.Accounts = append(keypair.Accounts, seedGeneratedWalletAccount1)
	keypair.LastUsedDerivationIndex = 0

	seedGeneratedWalletAccount2 := &Account{
		Address:   derivedAccs[accsmanagementcommon.CustomWalletPath1].Address(),
		KeyUID:    keypair.KeyUID,
		Wallet:    false,
		Chat:      false,
		Type:      AccountTypeSeed,
		Path:      accsmanagementcommon.CustomWalletPath1,
		PublicKey: types.Hex2Bytes(derivedAccs[accsmanagementcommon.CustomWalletPath1].PublicKeyHex()),
		Name:      "Seed Impo 1 Acc 2",
		Emoji:     "emoji-2",
		ColorID:   common.CustomizationColorPrimary,
		Hidden:    false,
		Clock:     0,
		Removed:   false,
		Operable:  AccountFullyOperable,
	}
	keypair.Accounts = append(keypair.Accounts, seedGeneratedWalletAccount2)
	keypair.LastUsedDerivationIndex = 1

	return
}

func GetSeedImportedKeypair2ForTest() (keypair *Keypair, acc *generator.Account, derivedAccs map[string]*generator.Account, err error) {
	acc, derivedAccs, err = generator.CreateAndDeriveAccountsFromMnemonic(testSeed2Mnemonic, []string{
		accsmanagementcommon.PathDefaultWalletAccount,
		accsmanagementcommon.CustomWalletPath1,
	}, "")
	if err != nil {
		return
	}

	keypair = &Keypair{
		KeyUID:      acc.KeyUID(),
		Name:        "Seed Imported 2",
		Type:        KeypairTypeSeed,
		DerivedFrom: acc.Address().Hex(),
	}

	seedGeneratedWalletAccount1 := &Account{
		Address:   derivedAccs[accsmanagementcommon.PathDefaultWalletAccount].Address(),
		KeyUID:    keypair.KeyUID,
		Wallet:    false,
		Chat:      false,
		Type:      AccountTypeSeed,
		Path:      accsmanagementcommon.PathDefaultWalletAccount,
		PublicKey: types.Hex2Bytes(derivedAccs[accsmanagementcommon.PathDefaultWalletAccount].PublicKeyHex()),
		Name:      "Seed Impo 2 Acc 1",
		Emoji:     "emoji-1",
		ColorID:   common.CustomizationColorPrimary,
		Hidden:    false,
		Clock:     0,
		Removed:   false,
		Operable:  AccountFullyOperable,
	}
	keypair.Accounts = append(keypair.Accounts, seedGeneratedWalletAccount1)
	keypair.LastUsedDerivationIndex = 0

	seedGeneratedWalletAccount2 := &Account{
		Address:   derivedAccs[accsmanagementcommon.CustomWalletPath1].Address(),
		KeyUID:    keypair.KeyUID,
		Wallet:    false,
		Chat:      false,
		Type:      AccountTypeSeed,
		Path:      "m/44'/60'/0'/0/1",
		PublicKey: types.Hex2Bytes("0x000000032"),
		Name:      "Seed Impo 2 Acc 2",
		Emoji:     "emoji-2",
		ColorID:   common.CustomizationColorPrimary,
		Hidden:    false,
		Clock:     0,
		Removed:   false,
		Operable:  AccountFullyOperable,
	}
	keypair.Accounts = append(keypair.Accounts, seedGeneratedWalletAccount2)
	keypair.LastUsedDerivationIndex = 1

	return
}

func GetPrivKeyImportedKeypairForTest() *Keypair {
	kp := &Keypair{
		KeyUID:      "0000000000000000000000000000000000000000000000000000000000000004",
		Name:        "Priv Key Imported",
		Type:        KeypairTypeKey,
		DerivedFrom: "", // no derived from for private key imported kp
	}

	privKeyWalletAccount := &Account{
		Address:   types.Address{0x41},
		KeyUID:    kp.KeyUID,
		Wallet:    false,
		Chat:      false,
		Type:      AccountTypeKey,
		Path:      "m",
		PublicKey: types.Hex2Bytes("0x000000041"),
		Name:      "Priv Key Impo Acc",
		Emoji:     "emoji-1",
		ColorID:   common.CustomizationColorPrimary,
		Hidden:    false,
		Clock:     0,
		Removed:   false,
		Operable:  AccountFullyOperable,
	}
	kp.Accounts = append(kp.Accounts, privKeyWalletAccount)

	return kp
}

func GetProfileKeycardForTest() *Keycard {
	profileKp, _, _, err := GetProfileKeypairForTest(true, true, true)
	if err != nil {
		panic(err)
	}
	keycard1Addresses := []types.Address{}
	for _, acc := range profileKp.Accounts {
		keycard1Addresses = append(keycard1Addresses, acc.Address)
	}
	return &Keycard{
		KeycardUID:        "00000000000000000000000000000001",
		KeycardName:       "Card01",
		KeycardLocked:     false,
		AccountsAddresses: keycard1Addresses,
		KeyUID:            profileKp.KeyUID,
		Position:          0,
	}
}

func GetKeycardForSeedImportedKeypair1ForTest() *Keycard {
	seed1Kp, _, _, err := GetSeedImportedKeypair1ForTest()
	if err != nil {
		panic(err)
	}
	keycard2Addresses := []types.Address{}
	for _, acc := range seed1Kp.Accounts {
		keycard2Addresses = append(keycard2Addresses, acc.Address)
	}
	return &Keycard{
		KeycardUID:        "00000000000000000000000000000002",
		KeycardName:       "Card02",
		KeycardLocked:     false,
		AccountsAddresses: keycard2Addresses,
		KeyUID:            seed1Kp.KeyUID,
		Position:          1,
	}
}

func GetKeycardForSeedImportedKeypair2ForTest() *Keycard {
	seed2Kp, _, _, err := GetSeedImportedKeypair2ForTest()
	if err != nil {
		panic(err)
	}
	keycard4Addresses := []types.Address{}
	for _, acc := range seed2Kp.Accounts {
		keycard4Addresses = append(keycard4Addresses, acc.Address)
	}
	return &Keycard{
		KeycardUID:        "00000000000000000000000000000003",
		KeycardName:       "Card03",
		KeycardLocked:     false,
		AccountsAddresses: keycard4Addresses,
		KeyUID:            seed2Kp.KeyUID,
		Position:          2,
	}
}

func Contains[T comparable](container []T, element T, isEqual func(T, T) bool) bool {
	for _, e := range container {
		if isEqual(e, element) {
			return true
		}
	}
	return false
}

func HaveSameElements[T comparable](a []T, b []T, isEqual func(T, T) bool) bool {
	for _, v := range a {
		if !Contains(b, v, isEqual) {
			return false
		}
	}
	return true
}

func SameAccounts(expected, real *Account) bool {
	return expected.Address == real.Address &&
		expected.KeyUID == real.KeyUID &&
		expected.Wallet == real.Wallet &&
		expected.Chat == real.Chat &&
		expected.Type == real.Type &&
		expected.Path == real.Path &&
		string(expected.PublicKey) == string(real.PublicKey) &&
		expected.Name == real.Name &&
		expected.Emoji == real.Emoji &&
		expected.ColorID == real.ColorID &&
		expected.Hidden == real.Hidden &&
		expected.Clock == real.Clock &&
		expected.Removed == real.Removed &&
		expected.ProdPreferredChainIDs == real.ProdPreferredChainIDs &&
		expected.TestPreferredChainIDs == real.TestPreferredChainIDs
}

func SameAccountsIncludingPosition(expected, real *Account) bool {
	return SameAccounts(expected, real) && expected.Position == real.Position
}

func SameAccountsWithDifferentOperable(expected, real *Account, expectedOperableValue AccountOperable) bool {
	return SameAccounts(expected, real) && real.Operable == expectedOperableValue
}

func SameKeypairs(expected, real *Keypair) bool {
	same := expected.KeyUID == real.KeyUID &&
		expected.Name == real.Name &&
		expected.Type == real.Type &&
		expected.DerivedFrom == real.DerivedFrom &&
		expected.LastUsedDerivationIndex == real.LastUsedDerivationIndex &&
		expected.Clock == real.Clock &&
		len(expected.Accounts) == len(real.Accounts)

	if same {
		for i := range expected.Accounts {
			found := false
			for j := range real.Accounts {
				if SameAccounts(expected.Accounts[i], real.Accounts[j]) {
					found = true
					break
				}
			}

			if !found {
				return false
			}
		}
	}

	return same
}

func SameKeypairsWithDifferentSyncedFrom(expected, real *Keypair, ignoreSyncedFrom bool, expectedSyncedFromValue string,
	expectedOperableValue AccountOperable) bool {
	same := expected.KeyUID == real.KeyUID &&
		expected.Name == real.Name &&
		expected.Type == real.Type &&
		expected.DerivedFrom == real.DerivedFrom &&
		expected.LastUsedDerivationIndex == real.LastUsedDerivationIndex &&
		expected.Clock == real.Clock &&
		len(expected.Accounts) == len(real.Accounts)

	if same && !ignoreSyncedFrom {
		same = same && real.SyncedFrom == expectedSyncedFromValue
	}

	if same {
		for i := range expected.Accounts {
			found := false
			for j := range real.Accounts {
				if SameAccountsWithDifferentOperable(expected.Accounts[i], real.Accounts[j], expectedOperableValue) {
					found = true
					break
				}
			}

			if !found {
				return false
			}
		}
	}

	return same
}

func SameKeycards(expected, real *Keycard) bool {
	same := expected.KeycardUID == real.KeycardUID &&
		expected.KeyUID == real.KeyUID &&
		expected.KeycardName == real.KeycardName &&
		expected.KeycardLocked == real.KeycardLocked &&
		expected.Position == real.Position &&
		len(expected.AccountsAddresses) == len(real.AccountsAddresses)

	if same {
		for i := range expected.AccountsAddresses {
			found := false
			for j := range real.AccountsAddresses {
				if expected.AccountsAddresses[i] == real.AccountsAddresses[j] {
					found = true
					break
				}
			}

			if !found {
				return false
			}
		}
	}

	return same
}
