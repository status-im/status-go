package server

import (
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
	"github.com/status-im/status-go/t/utils"
)

var (
	password = "password"
	keyUID   = "0xdeadbeef"
	expected = multiaccounts.Account{
		Name:      "cool account",
		KeyUID:    keyUID,
		ColorHash: [][]int{{4, 3}, {4, 0}, {4, 3}, {4, 0}},
		ColorID:   10,
		Images:    images.SampleIdentityImages(),
	}
	account1Hash = []byte{0x8f, 0xba, 0x35, 0x1, 0x2b, 0x9d, 0xad, 0xf0, 0x2d, 0x3c, 0x4d, 0x6, 0xb5, 0x22, 0x2, 0x47, 0xd4, 0x1c, 0xf4, 0x31, 0x2f, 0xb, 0x5b, 0x27, 0x5d, 0x43, 0x97, 0x58, 0x2d, 0xf0, 0xe1, 0xbe}
	account2Hash = []byte{0x9, 0xf8, 0x5c, 0xe9, 0x92, 0x96, 0x2d, 0x88, 0x2b, 0x8e, 0x42, 0x3f, 0xa4, 0x93, 0x6c, 0xad, 0xe9, 0xc0, 0x1b, 0x8a, 0x8, 0x8c, 0x5e, 0x7a, 0x84, 0xa2, 0xf, 0x9f, 0x77, 0x58, 0x2c, 0x2c}
)

func TestPayloadMarshallerSuite(t *testing.T) {
	suite.Run(t, new(PayloadMarshallerSuite))
}

type PayloadMarshallerSuite struct {
	suite.Suite

	teardown func()

	db1       *multiaccounts.Database
	db2       *multiaccounts.Database
	keystore1 string
	keystore2 string
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
	keyStoreDir, err := ioutil.TempDir(os.TempDir(), "accounts")
	require.NoError(t, err)

	emptyKeyStoreDir, err := ioutil.TempDir(os.TempDir(), "accounts_empty")
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
	var db1td func()
	var db2td func()
	var kstd func()

	pms.db1, db1td = setupTestDB(pms.T())
	pms.db2, db2td = setupTestDB(pms.T())
	pms.keystore1, pms.keystore2, kstd = makeKeystores(pms.T())
	pms.teardown = func() {
		db1td()
		db2td()
		kstd()
	}

	initKeys(pms.T(), pms.keystore1)
	err := pms.db1.SaveAccount(expected)
	pms.Require().NoError(err)
}

func (pms *PayloadMarshallerSuite) TearDownTest() {
	pms.teardown()
}

func (pms *PayloadMarshallerSuite) TestPayloadMarshaller_LoadPayloads() {
	// Make and LoadFromSource PairingPayloadRepository 1
	pm := NewPairingPayloadRepository(pms.db1)
	err := pm.LoadFromSource(pms.keystore1, keyUID, password)
	pms.Require().NoError(err)

	// TEST PairingPayloadRepository 1 LoadFromSource()
	pms.Require().Len(pm.keys, 2)
	pms.Require().Len(pm.keys[utils.GetAccount1PKFile()], 489)
	pms.Require().Len(pm.keys[utils.GetAccount2PKFile()], 489)

	h1 := sha256.New()
	h1.Write(pm.keys[utils.GetAccount1PKFile()])
	pms.Require().Exactly(account1Hash, h1.Sum(nil))

	h2 := sha256.New()
	h2.Write(pm.keys[utils.GetAccount2PKFile()])
	pms.Require().Exactly(account2Hash, h2.Sum(nil))

	pms.Require().Exactly(expected.ColorHash, pm.multiaccount.ColorHash)
	pms.Require().Exactly(expected.ColorID, pm.multiaccount.ColorID)
	pms.Require().Exactly(expected.Identicon, pm.multiaccount.Identicon)
	pms.Require().Exactly(expected.KeycardPairing, pm.multiaccount.KeycardPairing)
	pms.Require().Exactly(expected.KeyUID, pm.multiaccount.KeyUID)
	pms.Require().Exactly(expected.Name, pm.multiaccount.Name)
	pms.Require().Exactly(expected.Timestamp, pm.multiaccount.Timestamp)
	pms.Require().Len(pm.multiaccount.Images, 2)
	pms.Require().Equal(password, pm.password)
}

func (pms *PayloadMarshallerSuite) TestPayloadMarshaller_MarshalToProtobuf() {
	// Make and LoadFromSource PairingPayloadRepository 1
	ppr := NewPairingPayloadRepository(pms.db1)
	err := ppr.LoadFromSource(pms.keystore1, keyUID, password)
	pms.Require().NoError(err)

	// Make and Load PairingPayloadMarshaller 1
	ppm := NewPairingPayloadMarshaller()
	ppm.Load(ppr.PairingPayload)

	// TEST PairingPayloadMarshaller 1 MarshalToProtobuf()
	pb, err := ppm.MarshalToProtobuf()
	pms.Require().NoError(err)
	pms.Require().Len(pb, 1216)

	h := sha256.New()
	h.Write(pb)
	pms.Require().Exactly([]byte{0x70, 0xf2, 0xe5, 0x37, 0xff, 0x7d, 0x2d, 0x7b, 0x8a, 0x4b, 0x53, 0x1f, 0xfe, 0x3e, 0xea, 0x5e, 0x4d, 0xe1, 0xad, 0x44, 0xe8, 0x22, 0x5c, 0x84, 0x30, 0xd6, 0x75, 0x1a, 0xbd, 0x53, 0x59, 0xce}, h.Sum(nil))
}

func (pms *PayloadMarshallerSuite) TestPayloadMarshaller_UnmarshalProtobuf() {
	// Make and LoadFromSource PairingPayloadRepository 1
	ppr := NewPairingPayloadRepository(pms.db1)
	err := ppr.LoadFromSource(pms.keystore1, keyUID, password)
	pms.Require().NoError(err)

	// Make and Load PairingPayloadMarshaller 1
	ppm := NewPairingPayloadMarshaller()
	ppm.Load(ppr.PairingPayload)

	pb, err := ppm.MarshalToProtobuf()
	pms.Require().NoError(err)

	// Make PairingPayloadMarshaller 2
	ppm2 := NewPairingPayloadMarshaller()

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
	// Make and LoadFromSource PairingPayloadRepository 1
	ppr := NewPairingPayloadRepository(pms.db1)
	err := ppr.LoadFromSource(pms.keystore1, keyUID, password)
	pms.Require().NoError(err)

	// Make and Load PairingPayloadMarshaller 1
	ppm := NewPairingPayloadMarshaller()
	ppm.Load(ppr.PairingPayload)

	pb, err := ppm.MarshalToProtobuf()
	pms.Require().NoError(err)

	// Make PairingPayloadMarshaller 2
	ppm2 := NewPairingPayloadMarshaller()

	err = ppm2.UnmarshalProtobuf(pb)
	pms.Require().NoError(err)

	// Make and Load PairingPayloadRepository 2
	ppr2 := NewPairingPayloadRepository(pms.db2)
	ppr2.Load(ppm2.PairingPayload)

	err = ppr2.StoreToSource(pms.keystore2, password)
	pms.Require().NoError(err)

	// TEST PairingPayloadRepository 2 StoreToSource()
	keys := getFiles(pms.T(), pms.keystore2)

	pms.Require().Len(keys, 2)
	pms.Require().Len(keys[utils.GetAccount1PKFile()], 489)
	pms.Require().Len(keys[utils.GetAccount2PKFile()], 489)

	h1 := sha256.New()
	h1.Write(keys[utils.GetAccount1PKFile()])
	pms.Require().Exactly(account1Hash, h1.Sum(nil))

	h2 := sha256.New()
	h2.Write(keys[utils.GetAccount2PKFile()])
	pms.Require().Exactly(account2Hash, h2.Sum(nil))

	acc, err := pms.db2.GetAccount(keyUID)
	pms.Require().NoError(err)

	pms.Require().Exactly(expected.ColorHash, acc.ColorHash)
	pms.Require().Exactly(expected.ColorID, acc.ColorID)
	pms.Require().Exactly(expected.Identicon, acc.Identicon)
	pms.Require().Exactly(expected.KeycardPairing, acc.KeycardPairing)
	pms.Require().Exactly(expected.KeyUID, acc.KeyUID)
	pms.Require().Exactly(expected.Name, acc.Name)
	pms.Require().Exactly(expected.Timestamp, acc.Timestamp)
	pms.Require().Len(acc.Images, 2)
}
