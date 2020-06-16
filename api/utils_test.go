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
	secPk, _ := hex.DecodeString("04" +
			"261c55675e55ff25edb50b345cfb3a3f35f60712d251cbaaab97bd50054c6ebc" +
			"3cd4e22200c68daf7493e1f8da6a190a68a671e2d3977809612424c7c3888bc6")
	secPkt := append([]byte{0xe7, 0x01}, secPk...)

	bls12G1Pk, _ := hex.DecodeString("17f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac58" +
			"6c55e83ff97a1aeffb3af00adb22c6bb08b3f481e3aaa0f1a09e30ed741d8ae4" +
			"fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1")
	bls12G1Pkt := append([]byte{0xea, 0x01}, bls12G1Pk...)

	bls12G2Pk, _ := hex.DecodeString("13e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049" +
			"334cf11213945d57e5ac7d055d042b7e024aa2b2f08f0a91260805272dc51051" +
			"c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8" +
			"0606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab" +
			"3f370d275cec1da1aaa9075ff05f79be0ce5d527727d6e118cc9cdc6da2e351a" +
			"adfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801")
	bls12G2Pkt := append([]byte{0xeb, 0x01}, bls12G2Pk...)

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
			secPkt,
			"",
			fmt.Errorf("selected encoding not supported"),
		},
		{
			"valid key, no key type defined",
			"z",
			secPk,
			"",
			fmt.Errorf("unsupported public key type '4'"),
		},
		{
			"valid key, with base58 bitcoin encoding",
			"z",
			secPkt,
			"zQ3shPyZJnxZK4Bwyx9QsaksNKDYTPmpwPvGSjMYVHoXHeEgB",
			nil,
		},
		{
			"valid key, with traditional hex encoding",
			"0x",
			secPkt,
			"fe70102261c55675e55ff25edb50b345cfb3a3f35f60712d251cbaaab97bd50054c6ebc",
			nil,
		},
		{
			"valid secp256k1 key, with multiencoding hex encoding",
			"f",
			secPkt,
			"fe70102261c55675e55ff25edb50b345cfb3a3f35f60712d251cbaaab97bd50054c6ebc",
			nil,
		},
		{
			"valid bls12-381 g1 key, with multiencoding hex encoding",
			"f",
			bls12G1Pkt,
			"fea0197f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb",
			nil,
		},
		{
			"valid bls12-381 g1 key, with base58 bitcoin encoding",
			"z",
			bls12G1Pkt,
			"z3tEFUdV4D3tCMG6Fr1deVvt32DCS1Y4SxDGoELedXaMUdTdr5FfZvBnbK9bWMhAGj3RHk",
			nil,
		},
		{
			"valid bls12-381 g1 key, with no key type",
			"f",
			bls12G1Pk,
			"",
			fmt.Errorf("unsupported public key type '17'"),
		},
		{
			"valid bls12-381 g2 key, with multiencoding hex encoding",
			"f",
			bls12G2Pkt,
			"feb0193e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8",
			nil,
		},
		{
			"valid bls12-381 g2 key, with base58 bitcoin encoding",
			"z",
			bls12G2Pkt,
			"zUC77n3BqSWuoGMY7ut91NDoWzpithCd4GwPLAnv9fc7drWY4wBTvMX1y9eGSAuiBpktqGAocND2KXdu1HqNgrd6i3vCZKCLqZ3nQFaEA2FpTs7ZEChRpWReLvYyXNYUHvQjyKd",
			nil,
		},
		{
			"valid bls12-381 g2 key, with no key type",
			"f",
			bls12G2Pk,
			"",
			fmt.Errorf("unsupported public key type '13'"),
		},
	}

	for _, c := range cs {
		cpk, err := CompressPublicKey(c.Base, c.Key)

		require.Equal(t, c.Expected, cpk, c.Description)

		if c.Error != nil {
			require.EqualError(t, err, c.Error.Error(), c.Description)
			continue
		}

		require.NoError(t, err, c.Description)
	}
}

func TestDecompressPublicKey(t *testing.T) {
	secExpected := []byte{
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
	bls12G1Expected := []byte{
		0xea, 0x01, 0x17, 0xf1, 0xd3, 0xa7, 0x31, 0x97,
		0xd7, 0x94, 0x26, 0x95, 0x63, 0x8c, 0x4f, 0xa9,
		0xac, 0x0f, 0xc3, 0x68, 0x8c, 0x4f, 0x97, 0x74,
		0xb9, 0x05, 0xa1, 0x4e, 0x3a, 0x3f, 0x17, 0x1b,
		0xac, 0x58, 0x6c, 0x55, 0xe8, 0x3f, 0xf9, 0x7a,
		0x1a, 0xef, 0xfb, 0x3a, 0xf0, 0x0a, 0xdb, 0x22,
		0xc6, 0xbb, 0x08, 0xb3, 0xf4, 0x81, 0xe3, 0xaa,
		0xa0, 0xf1, 0xa0, 0x9e, 0x30, 0xed, 0x74, 0x1d,
		0x8a, 0xe4, 0xfc, 0xf5, 0xe0, 0x95, 0xd5, 0xd0,
		0x0a, 0xf6, 0x00, 0xdb, 0x18, 0xcb, 0x2c, 0x04,
		0xb3, 0xed, 0xd0, 0x3c, 0xc7, 0x44, 0xa2, 0x88,
		0x8a, 0xe4, 0x0c, 0xaa, 0x23, 0x29, 0x46, 0xc5,
		0xe7, 0xe1,
	}
	bls12G2Expected := []byte{
		0xeb, 0x01, 0x13, 0xe0, 0x2b, 0x60, 0x52, 0x71,
		0x9f, 0x60, 0x7d, 0xac, 0xd3, 0xa0, 0x88, 0x27,
		0x4f, 0x65, 0x59, 0x6b, 0xd0, 0xd0, 0x99, 0x20,
		0xb6, 0x1a, 0xb5, 0xda, 0x61, 0xbb, 0xdc, 0x7f,
		0x50, 0x49, 0x33, 0x4c, 0xf1, 0x12, 0x13, 0x94,
		0x5d, 0x57, 0xe5, 0xac, 0x7d, 0x05, 0x5d, 0x04,
		0x2b, 0x7e, 0x02, 0x4a, 0xa2, 0xb2, 0xf0, 0x8f,
		0x0a, 0x91, 0x26, 0x08, 0x05, 0x27, 0x2d, 0xc5,
		0x10, 0x51, 0xc6, 0xe4, 0x7a, 0xd4, 0xfa, 0x40,
		0x3b, 0x02, 0xb4, 0x51, 0x0b, 0x64, 0x7a, 0xe3,
		0xd1, 0x77, 0x0b, 0xac, 0x03, 0x26, 0xa8, 0x05,
		0xbb, 0xef, 0xd4, 0x80, 0x56, 0xc8, 0xc1, 0x21,
		0xbd, 0xb8, 0x06, 0x06, 0xc4, 0xa0, 0x2e, 0xa7,
		0x34, 0xcc, 0x32, 0xac, 0xd2, 0xb0, 0x2b, 0xc2,
		0x8b, 0x99, 0xcb, 0x3e, 0x28, 0x7e, 0x85, 0xa7,
		0x63, 0xaf, 0x26, 0x74, 0x92, 0xab, 0x57, 0x2e,
		0x99, 0xab, 0x3f, 0x37, 0x0d, 0x27, 0x5c, 0xec,
		0x1d, 0xa1, 0xaa, 0xa9, 0x07, 0x5f, 0xf0, 0x5f,
		0x79, 0xbe, 0x0c, 0xe5, 0xd5, 0x27, 0x72, 0x7d,
		0x6e, 0x11, 0x8c, 0xc9, 0xcd, 0xc6, 0xda, 0x2e,
		0x35, 0x1a, 0xad, 0xfd, 0x9b, 0xaa, 0x8c, 0xbd,
		0xd3, 0xa7, 0x6d, 0x42, 0x9a, 0x69, 0x51, 0x60,
		0xd1, 0x2c, 0x92, 0x3a, 0xc9, 0xcc, 0x3b, 0xac,
		0xa2, 0x89, 0xe1, 0x93, 0x54, 0x86, 0x08, 0xb8,
		0x28, 0x01,
	}

	fmt.Println(bls12G1Expected)
	fmt.Println(bls12G2Expected)

	cs := []struct {
		Description string
		Input       string
		Expected    []byte
		Error       error
	}{
		{
			"valid key with valid encoding type '0x'",
			"0xe70102261c55675e55ff25edb50b345cfb3a3f35f60712d251cbaaab97bd50054c6ebc",
			secExpected,
			nil,
		},
		{
			"valid key with valid encoding type 'f'",
			"fe70102261c55675e55ff25edb50b345cfb3a3f35f60712d251cbaaab97bd50054c6ebc",
			secExpected,
			nil,
		},
		{
			"valid key with valid encoding type 'z'",
			"zQ3shPyZJnxZK4Bwyx9QsaksNKDYTPmpwPvGSjMYVHoXHeEgB",
			secExpected,
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

		require.Exactly(t, c.Expected, key, c.Description)

		if c.Error != nil {
			require.EqualError(t, err, c.Error.Error(), c.Description)
			continue
		}

		require.NoError(t, err, c.Description)
	}
}
