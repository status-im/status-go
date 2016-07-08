package extkeys

import (
	"testing"
)

// TODO implement test vectors

func TestNewKey(t *testing.T) {
	seed, err := RandSeed()
	t.Log(len(seed))
	if err != nil {
		t.Error(err)
	}

	key, err := MasterKey(seed)
	if err != nil {
		t.Error(err)
	}
	t.Log(key)
}
