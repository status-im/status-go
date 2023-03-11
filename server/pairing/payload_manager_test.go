package pairing

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/t/utils"
)

var (
	password = "password"
	keyUID   = "0x6b9a74f33316e02479c33ed23cf16e0408dca3e1b9ab8f361630859543eb0d46"
	expected = multiaccounts.Account{
		Name:          "cool account",
		KeyUID:        keyUID,
		ColorHash:     multiaccounts.ColourHash{{4, 3}, {4, 0}, {4, 3}, {4, 0}},
		ColorID:       10,
		Images:        images.SampleIdentityImages(),
		KDFIterations: sqlite.ReducedKDFIterationsNumber,
	}
	account1Hash = []byte{0x8f, 0xba, 0x35, 0x1, 0x2b, 0x9d, 0xad, 0xf0, 0x2d, 0x3c, 0x4d, 0x6, 0xb5, 0x22, 0x2, 0x47, 0xd4, 0x1c, 0xf4, 0x31, 0x2f, 0xb, 0x5b, 0x27, 0x5d, 0x43, 0x97, 0x58, 0x2d, 0xf0, 0xe1, 0xbe}
	account2Hash = []byte{0x9, 0xf8, 0x5c, 0xe9, 0x92, 0x96, 0x2d, 0x88, 0x2b, 0x8e, 0x42, 0x3f, 0xa4, 0x93, 0x6c, 0xad, 0xe9, 0xc0, 0x1b, 0x8a, 0x8, 0x8c, 0x5e, 0x7a, 0x84, 0xa2, 0xf, 0x9f, 0x77, 0x58, 0x2c, 0x2c}
)

func TestPayloadMarshallerSuite(t *testing.T) {
	suite.Run(t, new(PayloadMarshallerSuite))
}

type PayloadMarshallerSuite struct {
	suite.Suite
	TestLoggerComponents

	teardown func()

	config1 *SenderConfig
	config2 *ReceiverConfig
}

func setupTestDB(t *testing.T) (*multiaccounts.Database, func()) {
	tmpfile, err := ioutil.TempFile("", "accounts-tests-")
	require.NoError(t, err)

	db, err := multiaccounts.InitializeDB(tmpfile.Name())
	require.NoError(t, err)

	return db, func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.Remove(tmpfile.Name()))
	}
}

func makeKeystores(t *testing.T) (string, string, func()) {
	keyStoreDir, err := os.MkdirTemp(os.TempDir(), "accounts")
	require.NoError(t, err)

	emptyKeyStoreDir, err := os.MkdirTemp(os.TempDir(), "accounts_empty")
	require.NoError(t, err)

	keyStoreDir = filepath.Join(keyStoreDir, "keystore", keyUID)
	// TODO test case where the keystore dir does not yet exist because the device is new
	emptyKeyStoreDir = filepath.Join(emptyKeyStoreDir, "keystore")

	err = os.MkdirAll(keyStoreDir, 0777)
	require.NoError(t, err)

	err = os.MkdirAll(emptyKeyStoreDir, 0777)
	require.NoError(t, err)

	return keyStoreDir, emptyKeyStoreDir, func() {
		os.RemoveAll(keyStoreDir)
		os.RemoveAll(emptyKeyStoreDir)
	}
}

func initKeys(t *testing.T, keyStoreDir string) {
	utils.Init()
	require.NoError(t, utils.ImportTestAccount(keyStoreDir, utils.GetAccount1PKFile()))
	require.NoError(t, utils.ImportTestAccount(keyStoreDir, utils.GetAccount2PKFile()))
}

func getFiles(t *testing.T, keyStorePath string) map[string][]byte {
	keys := make(map[string][]byte)

	fileWalker := func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fileInfo.IsDir() || filepath.Dir(path) != keyStorePath {
			return nil
		}

		rawKeyFile, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("invalid account key file: %v", err)
		}

		keys[fileInfo.Name()] = rawKeyFile
		return nil
	}

	err := filepath.Walk(keyStorePath, fileWalker)
	require.NoError(t, err)

	return keys
}

func (pms *PayloadMarshallerSuite) SetupTest() {
	pms.SetupLoggerComponents()

	db1, db1td := setupTestDB(pms.T())
	db2, db2td := setupTestDB(pms.T())
	keystore1, keystore2, kstd := makeKeystores(pms.T())
	pms.teardown = func() {
		db1td()
		db2td()
		kstd()
	}

	initKeys(pms.T(), keystore1)
	err := db1.SaveAccount(expected)
	pms.Require().NoError(err)

	pms.config1 = &SenderConfig{
		DB:           db1,
		KeystorePath: keystore1,
		KeyUID:       keyUID,
		Password:     password,
	}

	pms.config2 = &ReceiverConfig{
		DB:           db2,
		KeystorePath: keystore2,
	}
}

func (pms *PayloadMarshallerSuite) TearDownTest() {
	pms.teardown()
}

func (pms *PayloadMarshallerSuite) TestPayloadMarshaller_LoadPayloads() {
	// Make a Payload
	pp := new(AccountPayload)

	// Make and LoadFromSource PairingPayloadRepository 1
	ppr, err := NewAccountPayloadLoader(pp, pms.config1)
	pms.Require().NoError(err)
	err = ppr.LoadFromSource()
	pms.Require().NoError(err)

	// TEST PairingPayloadRepository 1 LoadFromSource()
	pms.Require().Len(ppr.keys, 2)
	pms.Require().Len(ppr.keys[utils.GetAccount1PKFile()], 489)
	pms.Require().Len(ppr.keys[utils.GetAccount2PKFile()], 489)

	h1 := sha256.New()
	h1.Write(ppr.keys[utils.GetAccount1PKFile()])
	pms.Require().Exactly(account1Hash, h1.Sum(nil))

	h2 := sha256.New()
	h2.Write(ppr.keys[utils.GetAccount2PKFile()])
	pms.Require().Exactly(account2Hash, h2.Sum(nil))

	pms.Require().Exactly(expected.ColorHash, ppr.multiaccount.ColorHash)
	pms.Require().Exactly(expected.ColorID, ppr.multiaccount.ColorID)
	pms.Require().Exactly(expected.Identicon, ppr.multiaccount.Identicon)
	pms.Require().Exactly(expected.KeycardPairing, ppr.multiaccount.KeycardPairing)
	pms.Require().Exactly(expected.KeyUID, ppr.multiaccount.KeyUID)
	pms.Require().Exactly(expected.Name, ppr.multiaccount.Name)
	pms.Require().Exactly(expected.Timestamp, ppr.multiaccount.Timestamp)
	pms.Require().Len(ppr.multiaccount.Images, 2)
	pms.Require().Equal(password, ppr.password)
}

func (pms *PayloadMarshallerSuite) TestPayloadMarshaller_MarshalToProtobuf() {
	// Make a Payload
	pp := new(AccountPayload)

	// Make and LoadFromSource PairingPayloadRepository 1
	ppr, err := NewAccountPayloadLoader(pp, pms.config1)
	pms.Require().NoError(err)
	err = ppr.LoadFromSource()
	pms.Require().NoError(err)

	// Make and Load PairingPayloadMarshaller 1
	ppm := NewPairingPayloadMarshaller(pp, pms.Logger)

	// TEST PairingPayloadMarshaller 1 MarshalProtobuf()
	pb, err := ppm.MarshalProtobuf()
	pms.Require().NoError(err)
	pms.Require().Len(pb, 1384)

	h := sha256.New()
	h.Write(pb)
	hashA := []byte{0xe5, 0x34, 0x2e, 0xf1, 0x81, 0x72, 0xab, 0xc3, 0xde, 0x54, 0xbc, 0x8e, 0xd8, 0x34, 0xbe, 0xab, 0xd, 0xe8, 0x84, 0x53, 0xa2, 0x14, 0x9b, 0xbe, 0xc5, 0xe5, 0xce, 0xa5, 0xe9, 0x6d, 0xbc, 0xdd}
	hashB := []byte{0x98, 0x2b, 0x3d, 0x8b, 0x7c, 0x6a, 0x3e, 0xdc, 0x3, 0xb1, 0xbf, 0xf1, 0x50, 0x15, 0xa5, 0x0, 0xa8, 0xba, 0xae, 0xf9, 0x38, 0xa8, 0x65, 0xd8, 0xf0, 0x93, 0xca, 0xbc, 0x47, 0x5d, 0x84, 0x23}

	// Because file-walk will pull files in an unpredictable order from a target dir
	// there are 2 potential valid hashes, because there are 2 key files in the test dir
	if !bytes.Equal(hashA, h.Sum(nil)) {
		pms.Require().Exactly(hashB, h.Sum(nil))
	}
}

func (pms *PayloadMarshallerSuite) TestPayloadMarshaller_UnmarshalProtobuf() {
	// Make a Payload
	pp := new(AccountPayload)

	// Make and LoadFromSource PairingPayloadRepository 1
	ppr, err := NewAccountPayloadLoader(pp, pms.config1)
	pms.Require().NoError(err)
	err = ppr.LoadFromSource()
	pms.Require().NoError(err)

	// Make and Load PairingPayloadMarshaller 1
	ppm := NewPairingPayloadMarshaller(pp, pms.Logger)

	pb, err := ppm.MarshalProtobuf()
	pms.Require().NoError(err)

	// Make a Payload
	pp2 := new(AccountPayload)

	// Make PairingPayloadMarshaller 2
	ppm2 := NewPairingPayloadMarshaller(pp2, pms.Logger)

	// TEST PairingPayloadMarshaller 2 is empty
	pms.Require().Nil(ppm2.keys)
	pms.Require().Nil(ppm2.multiaccount)
	pms.Require().Empty(ppm2.password)

	// TEST PairingPayloadMarshaller 2 UnmarshalProtobuf()
	err = ppm2.UnmarshalProtobuf(pb)
	pms.Require().NoError(err)

	pms.Require().Len(ppm2.keys, 2)
	pms.Require().Len(ppm2.keys[utils.GetAccount1PKFile()], 489)
	pms.Require().Len(ppm2.keys[utils.GetAccount2PKFile()], 489)

	h1 := sha256.New()
	h1.Write(ppm2.keys[utils.GetAccount1PKFile()])
	pms.Require().Exactly(account1Hash, h1.Sum(nil))

	h2 := sha256.New()
	h2.Write(ppm2.keys[utils.GetAccount2PKFile()])
	pms.Require().Exactly(account2Hash, h2.Sum(nil))

	pms.Require().Exactly(expected.ColorHash, ppm2.multiaccount.ColorHash)
	pms.Require().Exactly(expected.ColorID, ppm2.multiaccount.ColorID)
	pms.Require().Exactly(expected.Identicon, ppm2.multiaccount.Identicon)
	pms.Require().Exactly(expected.KeycardPairing, ppm2.multiaccount.KeycardPairing)
	pms.Require().Exactly(expected.KeyUID, ppm2.multiaccount.KeyUID)
	pms.Require().Exactly(expected.Name, ppm2.multiaccount.Name)
	pms.Require().Exactly(expected.Timestamp, ppm2.multiaccount.Timestamp)
	pms.Require().Len(ppm2.multiaccount.Images, 2)
	pms.Require().Equal(password, ppm2.password)
}

func (pms *PayloadMarshallerSuite) TestPayloadMarshaller_StorePayloads() {
	// Make a Payload
	pp := new(AccountPayload)

	// Make and LoadFromSource PairingPayloadRepository 1
	ppr, err := NewAccountPayloadLoader(pp, pms.config1)
	pms.Require().NoError(err)
	err = ppr.LoadFromSource()
	pms.Require().NoError(err)

	// Make and Load PairingPayloadMarshaller 1
	ppm := NewPairingPayloadMarshaller(pp, pms.Logger)

	pb, err := ppm.MarshalProtobuf()
	pms.Require().NoError(err)

	// Make a Payload
	pp2 := new(AccountPayload)

	// Make PairingPayloadMarshaller 2
	ppm2 := NewPairingPayloadMarshaller(pp2, pms.Logger)

	err = ppm2.UnmarshalProtobuf(pb)
	pms.Require().NoError(err)

	// Make and Load PairingPayloadRepository 2
	ppr2, err := NewAccountPayloadStorer(pp2, pms.config2)
	require.NoError(pms.T(), err)
	err = ppr2.StoreToSource()
	pms.Require().NoError(err)

	// TEST PairingPayloadRepository 2 StoreToSource()
	keys := getFiles(pms.T(), filepath.Join(pms.config2.KeystorePath, keyUID))

	pms.Require().Len(keys, 2)
	pms.Require().Len(keys[utils.GetAccount1PKFile()], 489)
	pms.Require().Len(keys[utils.GetAccount2PKFile()], 489)

	h1 := sha256.New()
	h1.Write(keys[utils.GetAccount1PKFile()])
	pms.Require().Exactly(account1Hash, h1.Sum(nil))

	h2 := sha256.New()
	h2.Write(keys[utils.GetAccount2PKFile()])
	pms.Require().Exactly(account2Hash, h2.Sum(nil))

	acc, err := pms.config2.DB.GetAccount(keyUID)
	pms.Require().NoError(err)

	pms.Require().Exactly(expected.ColorHash, acc.ColorHash)
	pms.Require().Exactly(expected.ColorID, acc.ColorID)
	pms.Require().Exactly(expected.Identicon, acc.Identicon)
	pms.Require().Exactly(expected.KeycardPairing, acc.KeycardPairing)
	pms.Require().Exactly(expected.KeyUID, acc.KeyUID)
	pms.Require().Exactly(expected.Name, acc.Name)
	pms.Require().Exactly(expected.Timestamp, acc.Timestamp)
	pms.Require().Len(acc.Images, 2)

	err = ppr2.storeKeys(ppr2.keystorePath)
	pms.Require().ErrorIs(err, ErrKeyFileAlreadyExists)
}

func (pms *PayloadMarshallerSuite) TestPayloadMarshaller_LockPayload() {
	AESKey := make([]byte, 32)
	_, err := rand.Read(AESKey)
	pms.Require().NoError(err)

	pm := NewMockPayloadMounter(AESKey)

	err = pm.Mount()
	pms.Require().NoError(err)

	toSend := pm.ToSend()
	pms.Len(toSend, 60)

	toSend2 := pm.ToSend()
	pms.Len(toSend2, 60)

	pm.LockPayload()

	toSend3 := pm.ToSend()
	pms.Nil(toSend3)
}
