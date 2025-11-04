package t

import (
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/crypto"
	"github.com/status-im/status-go/protocol/contacts"
)

func FakeContact(t *testing.T) *contacts.Contact {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	var contact *contacts.Contact
	err = gofakeit.Struct(&contact)
	require.NoError(t, err)

	contact.ID = contacts.ContactIDFromPublicKey(&key.PublicKey)

	return contact
}
