package api

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/personal"
	"github.com/status-im/status-go/t/utils"
)

func TestHashMessage(t *testing.T) {
	backend := NewGethStatusBackend()
	config, err := utils.MakeTestNodeConfig(params.StatusChainNetworkID)
	require.NoError(t, err)
	require.NoError(t, backend.AccountManager().InitKeystore(config.KeyStoreDir))
	err = backend.StartNode(config)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, backend.StopNode())
	}()

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	addr := crypto.PubkeyToAddress(key.PublicKey)

	scenarios := []struct {
		message        string
		expectedHash   string
		recoverMessage string
	}{
		{
			message:        "XYZ",
			expectedHash:   "634349abf2de883d23e8b46972896c7652a06670c990410d3436d9b44db09e6b",
			recoverMessage: fmt.Sprintf("0x%x", "XYZ"),
		},
		{
			message:        "0xXYZ",
			expectedHash:   "f9c57a8998c71a2c8d74d70abe6561838f0d6cb6d82bc85bd70afcc82368055c",
			recoverMessage: fmt.Sprintf("0x%x", "0xXYZ"),
		},
		{
			message:        "1122",
			expectedHash:   "3f07e02a153f02bdf97d77161746257626e9c39e4c3cf59896365fd1e6a9c7c3",
			recoverMessage: fmt.Sprintf("0x%x", "1122"),
		},
		{
			message:        "0x1122",
			expectedHash:   "86d79d0957efa9b7d91f1116e70d0ee934cb9cdeccefa07756aed2bee119a2f3",
			recoverMessage: "0x1122",
		},
	}

	for _, s := range scenarios {
		t.Run(s.message, func(t *testing.T) {
			hash, err := HashMessage(s.message)
			require.Nil(t, err)
			require.Equal(t, s.expectedHash, fmt.Sprintf("%x", hash))

			// simulate signature from external signer like a keycard
			sig, err := crypto.Sign(hash, key)
			require.NoError(t, err)
			sig[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper

			// check that the message was wrapped correctly before hashing it
			recParams := personal.RecoverParams{
				Message:   s.recoverMessage,
				Signature: fmt.Sprintf("0x%x", sig),
			}

			recoveredAddr, err := backend.Recover(recParams)
			require.NoError(t, err)
			assert.Equal(t, addr, recoveredAddr)
		})
	}
}

func TestCompressPublicKey(t *testing.T) {
	pk, _ := hex.DecodeString("e70104261c55675e55ff25edb50b345cfb3a3f35f60712d251cbaaab97bd50054c6ebc3cd4e22200c68daf7493e1f8da6a190a68a671e2d3977809612424c7c3888bc6")
	pk2, _ := hex.DecodeString("04261c55675e55ff25edb50b345cfb3a3f35f60712d251cbaaab97bd50054c6ebc3cd4e22200c68daf7493e1f8da6a190a68a671e2d3977809612424c7c3888bc6")

	cs := []struct {
		Description string
		Base        string
		Key         []byte
		Expected    string
		Error       error
	}{
		{
			"invalid key, with valid key type",
			"z",
			[]byte{0xe7, 0x1, 255, 66, 234},
			"",
			fmt.Errorf("invalid public key format, '[11111111 1000010 11101010]'"),
		},
		{
			"invalid key type, with invalid key",
			"z",
			[]byte{0xee, 255, 66, 234},
			"",
			fmt.Errorf("unsupported public key type '10BFEE'"),
		},
		{
			"invalid encoding type, with valid key",
			"p",
			pk,
			"",
			fmt.Errorf("selected encoding not supported"),
		},
		{
			"valid key, no key type defined",
			"z",
			pk2,
			"",
			fmt.Errorf("unsupported public key type '4'"),
		},
		{
			"valid key, with base58 bitcoin encoding",
			"z",
			pk,
			"zQ3shPyZJnxZK4Bwyx9QsaksNKDYTPmpwPvGSjMYVHoXHeEgB",
			nil,
		},
		{
			"valid key, with traditional hex encoding",
			"0x",
			pk,
			"fe70102261c55675e55ff25edb50b345cfb3a3f35f60712d251cbaaab97bd50054c6ebc",
			nil,
		},
		{
			"valid key, with multiencoding hex encoding",
			"f",
			pk,
			"fe70102261c55675e55ff25edb50b345cfb3a3f35f60712d251cbaaab97bd50054c6ebc",
			nil,
		},
	}

	for _, c := range cs {
		cpk, err := CompressPublicKey(c.Base, c.Key)

		if c.Error != nil {
			require.EqualError(t, err, c.Error.Error(), c.Description)
		}

		require.Equal(t, c.Expected, cpk, c.Description)
	}
}

func TestDecompressPublicKey(t *testing.T) {
	expected := []byte{
		0xe7, 0x01, 0x04, 0x26, 0x1c, 0x55, 0x67, 0x5e,
		0x55, 0xff, 0x25, 0xed, 0xb5, 0x0b, 0x34, 0x5c,
		0xfb, 0x3a, 0x3f, 0x35, 0xf6, 0x07, 0x12, 0xd2,
		0x51, 0xcb, 0xaa, 0xab, 0x97, 0xbd, 0x50, 0x05,
		0x4c, 0x6e, 0xbc, 0x3c, 0xd4, 0xe2, 0x22, 0x00,
		0xc6, 0x8d, 0xaf, 0x74, 0x93, 0xe1, 0xf8, 0xda,
		0x6a, 0x19, 0x0a, 0x68, 0xa6, 0x71, 0xe2, 0xd3,
		0x97, 0x78, 0x09, 0x61, 0x24, 0x24, 0xc7, 0xc3,
		0x88, 0x8b, 0xc6,
	}

	cs := []struct {
		Description string
		Input       string
		Expected    []byte
		Error       error
	}{
		{
			"valid key with valid encoding type '0x'",
			"0xe70102261c55675e55ff25edb50b345cfb3a3f35f60712d251cbaaab97bd50054c6ebc",
			expected,
			nil,
		},
		{
			"valid key with valid encoding type 'f'",
			"fe70102261c55675e55ff25edb50b345cfb3a3f35f60712d251cbaaab97bd50054c6ebc",
			expected,
			nil,
		},
		{
			"valid key with valid encoding type 'z'",
			"zQ3shPyZJnxZK4Bwyx9QsaksNKDYTPmpwPvGSjMYVHoXHeEgB",
			expected,
			nil,
		},
		{
			"valid key with mismatched encoding type 'f' instead of 'z'",
			"fQ3shPyZJnxZK4Bwyx9QsaksNKDYTPmpwPvGSjMYVHoXHeEgB",
			nil,
			fmt.Errorf("encoding/hex: invalid byte: U+0051 'Q'"),
		},
		{
			"valid key with no encoding type, in base58 encoding",
			"Q3shPyZJnxZK4Bwyx9QsaksNKDYTPmpwPvGSjMYVHoXHeEgB",
			nil,
			fmt.Errorf("selected encoding not supported"),
		},
	}

	for _, c := range cs {
		key, err := DecompressPublicKey(c.Input)

		if c.Error != nil {
			require.EqualError(t, err, c.Error.Error(), c.Description)
		}

		require.Exactly(t, c.Expected, key, c.Description)
	}
}
