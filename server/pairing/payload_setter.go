package pairing

type PayloadSetter interface {
	PayloadLocker
	PayloadResetter
	Encryptor

	// Receive accepts data from an inbound source into the PayloadSetter's state
	Receive(data []byte) error

	// Received returns a decrypted and parsed payload from an inbound source
	Received() []byte
}

/*
func (apr *AccountPayloadRepository) StoreToSource() error {
	keyUID := apr.multiaccount.KeyUID
	if apr.loggedInKeyUID != "" && apr.loggedInKeyUID != keyUID {
		return ErrLoggedInKeyUIDConflict
	}
	if apr.loggedInKeyUID == keyUID {
		// skip storing keys if user is logged in with the same key
		return nil
	}

	err := apr.validateKeys(apr.password)
	if err != nil {
		return err
	}

	if err = apr.storeKeys(apr.keystorePath); err != nil && err != ErrKeyFileAlreadyExists {
		return err
	}

	// skip storing multiaccount if key already exists
	if err == ErrKeyFileAlreadyExists {
		apr.exist = true
		apr.multiaccount, err = apr.multiaccountsDB.GetAccount(keyUID)
		if err != nil {
			return err
		}
		return nil
	}
	return apr.storeMultiAccount()
}

func (apr *AccountPayloadRepository) storeKeys(keyStorePath string) error {
	if keyStorePath == "" {
		return fmt.Errorf("keyStorePath can not be empty")
	}

	_, lastDir := filepath.Split(keyStorePath)

	// If lastDir == "keystore" we presume we need to create the rest of the keystore path
	// else we presume the provided keystore is valid
	if lastDir == "keystore" {
		if apr.multiaccount == nil || apr.multiaccount.KeyUID == "" {
			return fmt.Errorf("no known Key UID")
		}
		keyStorePath = filepath.Join(keyStorePath, apr.multiaccount.KeyUID)
		_, err := os.Stat(keyStorePath)
		if os.IsNotExist(err) {
			err := os.MkdirAll(keyStorePath, 0777)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			return ErrKeyFileAlreadyExists
		}
	}

	for name, data := range apr.keys {
		accountKey := new(keystore.EncryptedKeyJSONV3)
		if err := json.Unmarshal(data, &accountKey); err != nil {
			return fmt.Errorf("failed to read key file: %s", err)
		}

		if len(accountKey.Address) != 40 {
			return fmt.Errorf("account key address has invalid length '%s'", accountKey.Address)
		}

		err := ioutil.WriteFile(filepath.Join(keyStorePath, name), data, 0600)
		if err != nil {
			return err
		}
	}
	return nil
}

func (apr *AccountPayloadRepository) storeMultiAccount() error {
	apr.multiaccount.KDFIterations = apr.kdfIterations
	return apr.multiaccountsDB.SaveAccount(*apr.multiaccount)
}
*/
