package protocol

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"strings"
	"testing"

	"math/big"

	"github.com/stretchr/testify/suite"

	coretypes "github.com/status-im/status-go/eth-node/core/types"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/tt"
)

func padArray(bb []byte, size int) []byte {
	l := len(bb)
	if l == size {
		return bb
	}
	if l > size {
		return bb[l-size:]
	}
	tmp := make([]byte, size)
	copy(tmp[size-l:], bb)
	return tmp
}

type TransactionValidatorSuite struct {
	suite.Suite
}

func TestTransactionValidatorSuite(t *testing.T) {
	suite.Run(t, new(TransactionValidatorSuite))
}

func buildSignature(walletKey *ecdsa.PrivateKey, chatKey *ecdsa.PublicKey, hash string) ([]byte, error) {
	hashBytes, err := hex.DecodeString(hash[2:])
	if err != nil {
		return nil, err
	}
	chatKeyBytes := crypto.FromECDSAPub(chatKey)
	signatureMaterial := append(chatKeyBytes, hashBytes...)
	signatureMaterial = crypto.TextHash(signatureMaterial)
	signature, err := crypto.Sign(signatureMaterial, walletKey)
	if err != nil {
		return nil, err
	}
	signature[64] += 27
	return signature, nil
}

func buildData(fn string, to types.Address, value *big.Int) []byte {
	var data []byte
	addressBytes := make([]byte, 32)

	fnBytes, _ := hex.DecodeString(fn)
	copy(addressBytes[12:], to.Bytes())
	valueBytes := padArray(value.Bytes(), 32)

	data = append(data, fnBytes...)
	data = append(data, addressBytes...)
	data = append(data, valueBytes...)
	return data
}

func (s *TransactionValidatorSuite) TestValidateTransactions() {
	notTransferFunction := "a9059cbd"

	senderKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	senderWalletKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	myWalletKey1, err := crypto.GenerateKey()
	s.Require().NoError(err)
	myWalletKey2, err := crypto.GenerateKey()
	s.Require().NoError(err)

	senderAddress := crypto.PubkeyToAddress(senderWalletKey.PublicKey)
	myAddress1 := crypto.PubkeyToAddress(myWalletKey1.PublicKey)
	myAddress2 := crypto.PubkeyToAddress(myWalletKey2.PublicKey)

	db, err := openTestDB()
	s.Require().NoError(err)
	p := &sqlitePersistence{db: db}

	logger := tt.MustCreateTestLogger()
	validator := NewTransactionValidator([]types.Address{myAddress1, myAddress2}, p, nil, logger)

	contractString := "0x744d70fdbe2ba4cf95131626614a1763df805b9e"
	contractAddress := types.HexToAddress(contractString)

	defaultTransactionHash := "0x53edbe74408c2eeed4e5493b3aac0c006d8a14b140975f4306dd35f5e1d245bc"
	testCases := []struct {
		Name                     string
		Valid                    bool
		AccordingToSpec          bool
		Error                    bool
		Transaction              coretypes.Message
		OverrideSignatureChatKey *ecdsa.PublicKey
		OverrideTransactionHash  string
		Parameters               *CommandParameters
		WalletKey                *ecdsa.PrivateKey
		From                     *ecdsa.PublicKey
	}{
		{
			Name:            "valid eth transfer to any address",
			Valid:           true,
			AccordingToSpec: true,
			Transaction: coretypes.NewMessage(
				senderAddress,
				&myAddress1,
				1,
				big.NewInt(int64(23)),
				0,
				nil,
				nil,
				false,
			),
			Parameters: &CommandParameters{
				Value: "23",
			},
			WalletKey: senderWalletKey,
			From:      &senderKey.PublicKey,
		},
		{
			Name:            "valid eth transfer to specific address",
			Valid:           true,
			AccordingToSpec: true,
			Transaction: coretypes.NewMessage(
				senderAddress,
				&myAddress1,
				1,
				big.NewInt(int64(23)),
				0,
				nil,
				nil,
				false,
			),
			Parameters: &CommandParameters{
				Value:   "23",
				Address: strings.ToLower(myAddress1.Hex()),
			},
			WalletKey: senderWalletKey,
			From:      &senderKey.PublicKey,
		},
		{
			Name: "invalid eth transfer, not includes pk of the chat in signature",
			Transaction: coretypes.NewMessage(
				senderAddress,
				&myAddress1,
				1,
				big.NewInt(int64(23)),
				0,
				nil,
				nil,
				false,
			),
			Parameters: &CommandParameters{
				Value:   "23",
				Address: strings.ToLower(myAddress1.Hex()),
			},
			WalletKey:                senderWalletKey,
			OverrideSignatureChatKey: &senderWalletKey.PublicKey,
			From:                     &senderKey.PublicKey,
		},
		{
			Name: "invalid eth transfer, not signed with the wallet key",
			Transaction: coretypes.NewMessage(
				senderAddress,
				&myAddress1,
				1,
				big.NewInt(int64(23)),
				0,
				nil,
				nil,
				false,
			),
			Parameters: &CommandParameters{
				Value:   "23",
				Address: strings.ToLower(myAddress1.Hex()),
			},
			WalletKey: senderKey,
			From:      &senderKey.PublicKey,
		},
		{
			Name: "invalid eth transfer, wrong signature transaction hash",
			Transaction: coretypes.NewMessage(
				senderAddress,
				&myAddress1,
				1,
				big.NewInt(int64(23)),
				0,
				nil,
				nil,
				false,
			),
			OverrideTransactionHash: "0xdd9202df5e2f3611b5b6b716aef2a3543cc0bdd7506f50926e0869b83c8383b9",
			Parameters: &CommandParameters{
				Value: "23",
			},
			WalletKey: senderWalletKey,
			From:      &senderKey.PublicKey,
		},

		{
			Name: "invalid eth transfer, we own the wallet but not as specified",
			Transaction: coretypes.NewMessage(
				senderAddress,
				&myAddress1,
				1,
				big.NewInt(int64(23)),
				0,
				nil,
				nil,
				false,
			),
			Parameters: &CommandParameters{
				Value:   "23",
				Address: strings.ToLower(myAddress2.Hex()),
			},
			WalletKey: senderWalletKey,
			From:      &senderKey.PublicKey,
		},
		{
			Name: "invalid eth transfer, not our wallet",
			Transaction: coretypes.NewMessage(
				senderAddress,
				&senderAddress,
				1,
				big.NewInt(int64(23)),
				0,
				nil,
				nil,
				false,
			),
			Parameters: &CommandParameters{
				Value: "23",
			},
			WalletKey: senderWalletKey,
			From:      &senderKey.PublicKey,
		},
		{
			Name:  "valid eth transfer, but not according to spec, wrong amount",
			Valid: true,
			Transaction: coretypes.NewMessage(
				senderAddress,
				&myAddress1,
				1,
				big.NewInt(int64(20)),
				0,
				nil,
				nil,
				false,
			),
			Parameters: &CommandParameters{
				Value: "23",
			},
			WalletKey: senderWalletKey,
			From:      &senderKey.PublicKey,
		},
		{
			Name:            "valid token transfer to any address",
			Valid:           true,
			AccordingToSpec: true,
			Transaction: coretypes.NewMessage(
				senderAddress,
				&contractAddress,
				1,
				big.NewInt(int64(0)),
				0,
				nil,
				buildData(transferFunction, myAddress1, big.NewInt(int64(23))),
				false,
			),
			Parameters: &CommandParameters{
				Contract: contractString,
				Value:    "23",
			},
			WalletKey: senderWalletKey,
			From:      &senderKey.PublicKey,
		},
		{
			Name:            "valid token transfer to a specific address",
			Valid:           true,
			AccordingToSpec: true,
			Transaction: coretypes.NewMessage(
				senderAddress,
				&contractAddress,
				1,
				big.NewInt(int64(0)),
				0,
				nil,
				buildData(transferFunction, myAddress1, big.NewInt(int64(23))),
				false,
			),
			Parameters: &CommandParameters{
				Contract: contractString,
				Address:  strings.ToLower(myAddress1.Hex()),
				Value:    "23",
			},
			WalletKey: senderWalletKey,
			From:      &senderKey.PublicKey,
		},
		{
			Name:            "valid token transfer, not according to spec because of amount",
			Valid:           true,
			AccordingToSpec: false,
			Transaction: coretypes.NewMessage(
				senderAddress,
				&contractAddress,
				1,
				big.NewInt(int64(0)),
				0,
				nil,
				buildData(transferFunction, myAddress1, big.NewInt(int64(13))),
				false,
			),
			Parameters: &CommandParameters{
				Contract: contractString,
				Value:    "23",
			},
			WalletKey: senderWalletKey,
			From:      &senderKey.PublicKey,
		},
		{
			Name: "invalid token transfer, wrong contract",
			Transaction: coretypes.NewMessage(
				senderAddress,
				&senderAddress,
				1,
				big.NewInt(int64(0)),
				0,
				nil,
				buildData(transferFunction, myAddress1, big.NewInt(int64(23))),
				false,
			),
			Parameters: &CommandParameters{
				Contract: contractString,
				Address:  strings.ToLower(myAddress1.Hex()),
				Value:    "23",
			},
			WalletKey: senderWalletKey,
			From:      &senderKey.PublicKey,
		},

		{
			Name: "invalid token transfer, not an address I own",
			Transaction: coretypes.NewMessage(
				senderAddress,
				&contractAddress,
				1,
				big.NewInt(int64(0)),
				0,
				nil,
				buildData(transferFunction, myAddress1, big.NewInt(int64(23))),
				false,
			),
			Parameters: &CommandParameters{
				Contract: contractString,
				Address:  strings.ToLower(senderAddress.Hex()),
				Value:    "23",
			},
			WalletKey: senderWalletKey,
			From:      &senderKey.PublicKey,
		},

		{
			Name: "invalid token transfer, not the specified address",
			Transaction: coretypes.NewMessage(
				senderAddress,
				&contractAddress,
				1,
				big.NewInt(int64(0)),
				0,
				nil,
				buildData(transferFunction, myAddress2, big.NewInt(int64(23))),
				false,
			),
			Parameters: &CommandParameters{
				Contract: contractString,
				Address:  strings.ToLower(myAddress1.Hex()),
				Value:    "23",
			},
			WalletKey: senderWalletKey,
			From:      &senderKey.PublicKey,
		},
		{
			Name: "invalid token transfer, wrong fn",
			Transaction: coretypes.NewMessage(
				senderAddress,
				&contractAddress,
				1,
				big.NewInt(int64(0)),
				0,
				nil,
				buildData(notTransferFunction, myAddress1, big.NewInt(int64(23))),
				false,
			),
			Parameters: &CommandParameters{
				Contract: contractString,
				Value:    "23",
			},
			WalletKey: senderWalletKey,
			From:      &senderKey.PublicKey,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.Name, func() {
			tc.Parameters.TransactionHash = defaultTransactionHash
			signatureTransactionHash := defaultTransactionHash
			signatureChatKey := tc.From
			if tc.OverrideTransactionHash != "" {
				signatureTransactionHash = tc.OverrideTransactionHash
			}
			if tc.OverrideSignatureChatKey != nil {
				signatureChatKey = tc.OverrideSignatureChatKey
			}
			signature, err := buildSignature(tc.WalletKey, signatureChatKey, signatureTransactionHash)
			s.Require().NoError(err)
			tc.Parameters.Signature = signature

			response, err := validator.validateTransaction(context.Background(), tc.Transaction, tc.Parameters, tc.From)
			if tc.Error {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			s.Equal(tc.AccordingToSpec, response.AccordingToSpec)
			s.Equal(tc.Valid, response.Valid)
		})
	}

}
