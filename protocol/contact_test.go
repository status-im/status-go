package protocol

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/common"
)

func TestMarshalContactJSON(t *testing.T) {
	contact := &Contact{}
	id, err := crypto.GenerateKey()
	require.NoError(t, err)
	contact.ID = common.PubkeyToHex(&id.PublicKey)

	encodedContact, err := json.Marshal(contact)

	require.NoError(t, err)
	require.True(t, strings.Contains(string(encodedContact), "compressedKey\":\"zQ"))
}
