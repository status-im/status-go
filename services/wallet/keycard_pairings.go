package wallet

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
)

type KeycardPairings struct {
	pairingsFile string
}

type KeycardPairing struct {
	Key   string `json:"key"`
	Index int    `json:"index"`
}

func NewKeycardPairings() *KeycardPairings {
	return &KeycardPairings{}
}

func (kp *KeycardPairings) SetKeycardPairingsFile(filePath string) {
	kp.pairingsFile = filePath
}

func (kp *KeycardPairings) GetPairingsJSONFileContent() ([]byte, error) {
	_, err := os.Stat(kp.pairingsFile)
	if os.IsNotExist(err) {
		return []byte{}, nil
	}

	return ioutil.ReadFile(kp.pairingsFile)
}

func (kp *KeycardPairings) SetPairingsJSONFileContent(content []byte) error {
	if len(content) == 0 {
		// Nothing to write
		return nil
	}
	_, err := os.Stat(kp.pairingsFile)
	if os.IsNotExist(err) {
		dir, _ := filepath.Split(kp.pairingsFile)
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			return err
		}
	}

	return ioutil.WriteFile(kp.pairingsFile, content, 0600)
}

func (kp *KeycardPairings) GetPairings() (map[string]KeycardPairing, error) {
	content, err := kp.GetPairingsJSONFileContent()
	if err != nil {
		return nil, err
	}

	if len(content) == 0 {
		return nil, os.ErrNotExist
	}

	pairings := make(map[string]KeycardPairing)
	err = json.Unmarshal(content, &pairings)
	if err != nil {
		return nil, err
	}

	return pairings, nil
}
