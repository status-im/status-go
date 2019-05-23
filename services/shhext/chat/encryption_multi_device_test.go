package chat

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/suite"
	"os"
	"sort"
	"testing"
)

const (
	aliceUser = "alice"
	bobUser   = "bob"
)

func TestEncryptionServiceMultiDeviceSuite(t *testing.T) {
	suite.Run(t, new(EncryptionServiceMultiDeviceSuite))
}

type serviceAndKey struct {
	encryptionServices []*EncryptionService
	key                *ecdsa.PrivateKey
}

type EncryptionServiceMultiDeviceSuite struct {
	suite.Suite
	services map[string]*serviceAndKey
}

func setupUser(user string, s *EncryptionServiceMultiDeviceSuite, n int) error {
	key, err := crypto.GenerateKey()
	if err != nil {
		return err
	}

	s.services[user] = &serviceAndKey{
		key:                key,
		encryptionServices: make([]*EncryptionService, n),
	}

	for i := 0; i < n; i++ {
		installationID := fmt.Sprintf("%s%d", user, i+1)
		dbPath := fmt.Sprintf("/tmp/%s.db", installationID)

		os.Remove(dbPath)

		persistence, err := NewSQLLitePersistence(dbPath, "key")
		if err != nil {
			return err
		}

		config := DefaultEncryptionServiceConfig(installationID)
		config.MaxInstallations = n - 1

		s.services[user].encryptionServices[i] = NewEncryptionService(persistence, config)

	}

	return nil
}

func (s *EncryptionServiceMultiDeviceSuite) SetupTest() {
	s.services = make(map[string]*serviceAndKey)
	err := setupUser(aliceUser, s, 4)
	s.Require().NoError(err)

	err = setupUser(bobUser, s, 4)
	s.Require().NoError(err)
}

func (s *EncryptionServiceMultiDeviceSuite) TestProcessPublicBundle() {
	aliceKey := s.services[aliceUser].key

	alice2Bundle, err := s.services[aliceUser].encryptionServices[1].CreateBundle(aliceKey)
	s.Require().NoError(err)

	alice2Identity, err := ExtractIdentity(alice2Bundle)
	s.Require().NoError(err)

	alice3Bundle, err := s.services[aliceUser].encryptionServices[2].CreateBundle(aliceKey)
	s.Require().NoError(err)

	alice3Identity, err := ExtractIdentity(alice2Bundle)
	s.Require().NoError(err)

	// Add alice2 bundle
	response, err := s.services[aliceUser].encryptionServices[0].ProcessPublicBundle(aliceKey, alice2Bundle)
	s.Require().NoError(err)
	s.Require().Equal(IdentityAndIDPair{alice2Identity, "alice2"}, response[0])

	// Add alice3 bundle
	response, err = s.services[aliceUser].encryptionServices[0].ProcessPublicBundle(aliceKey, alice3Bundle)
	s.Require().NoError(err)
	s.Require().Equal(IdentityAndIDPair{alice3Identity, "alice3"}, response[0])

	// No installation is enabled
	alice1MergedBundle1, err := s.services[aliceUser].encryptionServices[0].CreateBundle(aliceKey)
	s.Require().NoError(err)

	s.Require().Equal(1, len(alice1MergedBundle1.GetSignedPreKeys()))
	s.Require().NotNil(alice1MergedBundle1.GetSignedPreKeys()["alice1"])

	// We enable the installations
	err = s.services[aliceUser].encryptionServices[0].EnableInstallation(&aliceKey.PublicKey, "alice2")
	s.Require().NoError(err)

	err = s.services[aliceUser].encryptionServices[0].EnableInstallation(&aliceKey.PublicKey, "alice3")
	s.Require().NoError(err)

	alice1MergedBundle2, err := s.services[aliceUser].encryptionServices[0].CreateBundle(aliceKey)
	s.Require().NoError(err)

	// We get back a bundle with all the installations
	s.Require().Equal(3, len(alice1MergedBundle2.GetSignedPreKeys()))
	s.Require().NotNil(alice1MergedBundle2.GetSignedPreKeys()["alice1"])
	s.Require().NotNil(alice1MergedBundle2.GetSignedPreKeys()["alice2"])
	s.Require().NotNil(alice1MergedBundle2.GetSignedPreKeys()["alice3"])

	response, err = s.services[aliceUser].encryptionServices[0].ProcessPublicBundle(aliceKey, alice1MergedBundle2)
	s.Require().NoError(err)
	sort.Slice(response, func(i, j int) bool {
		return response[i][1] < response[j][1]
	})
	// We only get back installationIDs not equal to us
	s.Require().Equal(2, len(response))
	s.Require().Equal(IdentityAndIDPair{alice2Identity, "alice2"}, response[0])
	s.Require().Equal(IdentityAndIDPair{alice2Identity, "alice3"}, response[1])

	// We disable the installations
	err = s.services[aliceUser].encryptionServices[0].DisableInstallation(&aliceKey.PublicKey, "alice2")
	s.Require().NoError(err)

	alice1MergedBundle3, err := s.services[aliceUser].encryptionServices[0].CreateBundle(aliceKey)
	s.Require().NoError(err)

	// We get back a bundle with all the installations
	s.Require().Equal(2, len(alice1MergedBundle3.GetSignedPreKeys()))
	s.Require().NotNil(alice1MergedBundle3.GetSignedPreKeys()["alice1"])
	s.Require().NotNil(alice1MergedBundle3.GetSignedPreKeys()["alice3"])
}

func (s *EncryptionServiceMultiDeviceSuite) TestProcessPublicBundleOutOfOrder() {
	aliceKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// Alice1 creates a bundle
	alice1Bundle, err := s.services[aliceUser].encryptionServices[0].CreateBundle(aliceKey)
	s.Require().NoError(err)

	// Alice2 Receives the bundle
	_, err = s.services[aliceUser].encryptionServices[1].ProcessPublicBundle(aliceKey, alice1Bundle)
	s.Require().NoError(err)

	// Alice2 Creates a Bundle
	_, err = s.services[aliceUser].encryptionServices[1].CreateBundle(aliceKey)
	s.Require().NoError(err)

	// We enable the installation
	err = s.services[aliceUser].encryptionServices[1].EnableInstallation(&aliceKey.PublicKey, "alice1")
	s.Require().NoError(err)

	// It should contain both bundles
	alice2MergedBundle1, err := s.services[aliceUser].encryptionServices[1].CreateBundle(aliceKey)
	s.Require().NoError(err)

	s.Require().NotNil(alice2MergedBundle1.GetSignedPreKeys()["alice1"])
	s.Require().NotNil(alice2MergedBundle1.GetSignedPreKeys()["alice2"])
}

func pairDevices(s *serviceAndKey, target int) error {
	device := s.encryptionServices[target]
	for i := 0; i < len(s.encryptionServices); i++ {
		b, err := s.encryptionServices[i].CreateBundle(s.key)

		if err != nil {
			return err
		}

		_, err = device.ProcessPublicBundle(s.key, b)
		if err != nil {
			return err
		}

		err = device.EnableInstallation(&s.key.PublicKey, s.encryptionServices[i].config.InstallationID)
		if err != nil {
			return nil
		}
	}
	return nil
}

func (s *EncryptionServiceMultiDeviceSuite) TestMaxDevices() {
	err := pairDevices(s.services[aliceUser], 0)
	s.Require().NoError(err)
	alice1 := s.services[aliceUser].encryptionServices[0]
	bob1 := s.services[bobUser].encryptionServices[0]
	aliceKey := s.services[aliceUser].key
	bobKey := s.services[bobUser].key

	// Check bundle is ok
	// No installation is enabled
	aliceBundle, err := alice1.CreateBundle(aliceKey)
	s.Require().NoError(err)

	// Check all installations are correctly working, and that the oldest device is not there
	preKeys := aliceBundle.GetSignedPreKeys()
	s.Require().Equal(3, len(preKeys))
	s.Require().NotNil(preKeys["alice1"])
	// alice2 being the oldest device is rotated out, as we reached the maximum
	s.Require().Nil(preKeys["alice2"])
	s.Require().NotNil(preKeys["alice3"])
	s.Require().NotNil(preKeys["alice4"])

	// We propagate this to bob
	_, err = bob1.ProcessPublicBundle(bobKey, aliceBundle)
	s.Require().NoError(err)

	// Bob sends a message to alice
	payload, _, err := bob1.EncryptPayload(&aliceKey.PublicKey, bobKey, []byte("test"))
	s.Require().NoError(err)
	s.Require().Equal(3, len(payload))
	s.Require().NotNil(payload["alice1"])
	s.Require().NotNil(payload["alice3"])
	s.Require().NotNil(payload["alice4"])

	// We disable the last installation
	err = s.services[aliceUser].encryptionServices[0].DisableInstallation(&aliceKey.PublicKey, "alice4")
	s.Require().NoError(err)

	// We check the bundle is updated
	aliceBundle, err = alice1.CreateBundle(aliceKey)
	s.Require().NoError(err)

	// Check all installations are there
	preKeys = aliceBundle.GetSignedPreKeys()
	s.Require().Equal(3, len(preKeys))
	s.Require().NotNil(preKeys["alice1"])
	s.Require().NotNil(preKeys["alice2"])
	s.Require().NotNil(preKeys["alice3"])
	// alice4 is disabled at this point, alice2 is back in
	s.Require().Nil(preKeys["alice4"])

	// We propagate this to bob
	_, err = bob1.ProcessPublicBundle(bobKey, aliceBundle)
	s.Require().NoError(err)

	// Bob sends a message to alice
	payload, _, err = bob1.EncryptPayload(&aliceKey.PublicKey, bobKey, []byte("test"))
	s.Require().NoError(err)
	s.Require().Equal(3, len(payload))
	s.Require().NotNil(payload["alice1"])
	s.Require().NotNil(payload["alice2"])
	s.Require().NotNil(payload["alice3"])
}
