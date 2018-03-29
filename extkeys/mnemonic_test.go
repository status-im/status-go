package extkeys

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

type VectorsFile struct {
	Data    map[string][][6]string
	vectors []*Vector
}

type Vector struct {
	language, password, input, mnemonic, seed, xprv string
}

func TestNewMnemonic(t *testing.T) {
	m1 := NewMnemonic()
	if m1.salt != Salt {
		t.Errorf("expected default salt, got: %q", m1.salt)
	}
}

func TestMnemonic_WordList(t *testing.T) {
	m := NewMnemonic()
	_, err := m.WordList(EnglishLanguage)
	if err != nil {
		t.Errorf("expected WordList to return WordList without errors, got: %s", err)
	}

	indexes := []Language{-1, Language(len(m.wordLists))}
	for _, index := range indexes {
		_, err := m.WordList(index)
		if err == nil {
			t.Errorf("expected WordList to return an error with index %d", index)
		}
	}
}

// TestMnemonicPhrase
func TestMnemonicPhrase(t *testing.T) {

	mnemonic := NewMnemonic()

	// test strength validation
	strengths := []entropyStrength{127, 129, 257}
	for _, s := range strengths {
		_, err := mnemonic.MnemonicPhrase(s, EnglishLanguage)
		if err != ErrInvalidEntropyStrength {
			t.Errorf("Entropy strength '%d' should be invalid", s)
		}
	}

	// test mnemonic generation
	t.Log("Test mnemonic generation:")
	for _, language := range mnemonic.AvailableLanguages() {
		phrase, err := mnemonic.MnemonicPhrase(EntropyStrength128, language)
		t.Logf("Mnemonic (%s): %s", Languages[language], phrase)

		if err != nil {
			t.Errorf("Test failed: could not create seed: %s", err)
		}

		if !mnemonic.ValidMnemonic(phrase, language) {
			t.Error("Seed is not valid Mnenomic")
		}
	}

	// run against test vectors
	vectorsFile, err := LoadVectorsFile("mnemonic_vectors.json")
	if err != nil {
		t.Error(err)
	}

	t.Log("Test against pre-computed seed vectors:")
	stats := map[string]int{}
	for _, vector := range vectorsFile.vectors {
		stats[vector.language]++
		mnemonic := NewMnemonic()
		seed := mnemonic.MnemonicSeed(vector.mnemonic, vector.password)
		if fmt.Sprintf("%x", seed) != vector.seed {
			t.Errorf("Test failed (%s): incorrect seed (%x) generated (expected: %s)", vector.language, seed, vector.seed)
			return
		}
		//t.Logf("Test passed: correct seed (%x) generated (expected: %s)", seed, vector.seed)

		rootKey, err := NewMaster(seed)
		if err != nil {
			t.Error(err)
		}

		if rootKey.String() != vector.xprv {
			t.Errorf("Test failed (%s): incorrect xprv (%s) generated (expected: %s)", vector.language, rootKey, vector.xprv)
		}
	}
	for language, count := range stats {
		t.Logf("[%s]: %d tests completed", language, count)
	}
}

func LoadVectorsFile(path string) (*VectorsFile, error) {
	fp, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Test failed: cannot open vectors file: %s", err)
	}

	var vectorsFile VectorsFile
	if err := json.NewDecoder(fp).Decode(&vectorsFile); err != nil {
		return nil, fmt.Errorf("Test failed: cannot parse vectors file: %s", err)
	}

	// load data into Vector structs
	for language, data := range vectorsFile.Data {
		for _, item := range data {
			vectorsFile.vectors = append(vectorsFile.vectors, &Vector{language, item[0], item[1], item[2], item[3], item[4]})
		}
	}

	return &vectorsFile, nil
}

func (v *Vector) String() string {
	return fmt.Sprintf("{password: %s, input: %s, mnemonic: %s, seed: %s, xprv: %s}",
		v.password, v.input, v.mnemonic, v.seed, v.xprv)
}
