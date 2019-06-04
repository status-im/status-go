package chat

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/suite"
	"os"
	"sort"
	"testing"

	"github.com/status-im/status-go/services/shhext/chat/multidevice"
	"github.com/status-im/status-go/services/shhext/chat/sharedsecret"
)

const (
	aliceUser = "alice"
	bobUser   = "bob"
)

func TestEncryptionServiceMultiDeviceSuite(t *testing.T) {
	suite.Run(t, new(EncryptionServiceMultiDeviceSuite))
}

type serviceAndKey struct {
	services []*ProtocolService
	key      *ecdsa.PrivateKey
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
		key:      key,
		services: make([]*ProtocolService, n),
	}

	for i := 0; i < n; i++ {
		installationID := fmt.Sprintf("%s%d", user, i+1)
		dbPath := fmt.Sprintf("/tmp/%s.db", installationID)

		os.Remove(dbPath)

		persistence, err := NewSQLLitePersistence(dbPath, "key")
		if err != nil {
			return err
		}
		// Initialize sharedsecret
		multideviceConfig := &multidevice.Config{
			MaxInstallations: n - 1,
			InstallationID:   installationID,
			ProtocolVersion:  1,
		}

		sharedSecretService := sharedsecret.NewService(persistence.GetSharedSecretStorage())
		multideviceService := multidevice.New(multideviceConfig, persistence.GetMultideviceStorage())

		protocol := NewProtocolService(
			NewEncryptionService(
				persistence,
				DefaultEncryptionServiceConfig(installationID)),
			sharedSecretService,
			multideviceService,
			func(s []*multidevice.IdentityAndID) {},
			func(s []*sharedsecret.Secret) {},
		)

		s.services[user].services[i] = protocol

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

	alice2Bundle, err := s.services[aliceUser].services[1].GetBundle(aliceKey)
	s.Require().NoError(err)

	alice2IdentityPK, err := ExtractIdentity(alice2Bundle)
	s.Require().NoError(err)

	alice2Identity := fmt.Sprintf("0x%x", crypto.FromECDSAPub(alice2IdentityPK))

	alice3Bundle, err := s.services[aliceUser].services[2].GetBundle(aliceKey)
	s.Require().NoError(err)

	alice3IdentityPK, err := ExtractIdentity(alice2Bundle)
	s.Require().NoError(err)

	alice3Identity := fmt.Sprintf("0x%x", crypto.FromECDSAPub(alice3IdentityPK))

	// Add alice2 bundle
	response, err := s.services[aliceUser].services[0].ProcessPublicBundle(aliceKey, alice2Bundle)
	s.Require().NoError(err)
	s.Require().Equal(multidevice.IdentityAndID{alice2Identity, "alice2"}, *response[0])

	// Add alice3 bundle
	response, err = s.services[aliceUser].services[0].ProcessPublicBundle(aliceKey, alice3Bundle)
	s.Require().NoError(err)
	s.Require().Equal(multidevice.IdentityAndID{alice3Identity, "alice3"}, *response[0])

	// No installation is enabled
	alice1MergedBundle1, err := s.services[aliceUser].services[0].GetBundle(aliceKey)
	s.Require().NoError(err)

	s.Require().Equal(1, len(alice1MergedBundle1.GetSignedPreKeys()))
	s.Require().NotNil(alice1MergedBundle1.GetSignedPreKeys()["alice1"])

	// We enable the installations
	err = s.services[aliceUser].services[0].EnableInstallation(&aliceKey.PublicKey, "alice2")
	s.Require().NoError(err)

	err = s.services[aliceUser].services[0].EnableInstallation(&aliceKey.PublicKey, "alice3")
	s.Require().NoError(err)

	alice1MergedBundle2, err := s.services[aliceUser].services[0].GetBundle(aliceKey)
	s.Require().NoError(err)

	// We get back a bundle with all the installations
	s.Require().Equal(3, len(alice1MergedBundle2.GetSignedPreKeys()))
	s.Require().NotNil(alice1MergedBundle2.GetSignedPreKeys()["alice1"])
	s.Require().NotNil(alice1MergedBundle2.GetSignedPreKeys()["alice2"])
	s.Require().NotNil(alice1MergedBundle2.GetSignedPreKeys()["alice3"])

	response, err = s.services[aliceUser].services[0].ProcessPublicBundle(aliceKey, alice1MergedBundle2)
	s.Require().NoError(err)
	sort.Slice(response, func(i, j int) bool {
		return response[i].ID < response[j].ID
	})
	// We only get back installationIDs not equal to us
	s.Require().Equal(2, len(response))
	s.Require().Equal(multidevice.IdentityAndID{alice2Identity, "alice2"}, *response[0])
	s.Require().Equal(multidevice.IdentityAndID{alice2Identity, "alice3"}, *response[1])

	// We disable the installations
	err = s.services[aliceUser].services[0].DisableInstallation(&aliceKey.PublicKey, "alice2")
	s.Require().NoError(err)

	alice1MergedBundle3, err := s.services[aliceUser].services[0].GetBundle(aliceKey)
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
	alice1Bundle, err := s.services[aliceUser].services[0].GetBundle(aliceKey)
	s.Require().NoError(err)

	// Alice2 Receives the bundle
	_, err = s.services[aliceUser].services[1].ProcessPublicBundle(aliceKey, alice1Bundle)
	s.Require().NoError(err)

	// Alice2 Creates a Bundle
	_, err = s.services[aliceUser].services[1].GetBundle(aliceKey)
	s.Require().NoError(err)

	// We enable the installation
	err = s.services[aliceUser].services[1].EnableInstallation(&aliceKey.PublicKey, "alice1")
	s.Require().NoError(err)

	// It should contain both bundles
	alice2MergedBundle1, err := s.services[aliceUser].services[1].GetBundle(aliceKey)
	s.Require().NoError(err)

	s.Require().NotNil(alice2MergedBundle1.GetSignedPreKeys()["alice1"])
	s.Require().NotNil(alice2MergedBundle1.GetSignedPreKeys()["alice2"])
}

func pairDevices(s *serviceAndKey, target int) error {
	device := s.services[target]
	for i := 0; i < len(s.services); i++ {
		b, err := s.services[i].GetBundle(s.key)

		if err != nil {
			return err
		}

		_, err = device.ProcessPublicBundle(s.key, b)
		if err != nil {
			return err
		}

		err = device.EnableInstallation(&s.key.PublicKey, s.services[i].encryption.config.InstallationID)
		if err != nil {
			return nil
		}
	}
	return nil
}

func (s *EncryptionServiceMultiDeviceSuite) TestMaxDevices() {
	err := pairDevices(s.services[aliceUser], 0)
	s.Require().NoError(err)
	alice1 := s.services[aliceUser].services[0]
	bob1 := s.services[bobUser].services[0]
	aliceKey := s.services[aliceUser].key
	bobKey := s.services[bobUser].key

	// Check bundle is ok
	// No installation is enabled
	aliceBundle, err := alice1.GetBundle(aliceKey)
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
	msg, err := bob1.BuildDirectMessage(bobKey, &aliceKey.PublicKey, []byte("test"))
	s.Require().NoError(err)
	payload := msg.Message.GetDirectMessage()
	s.Require().Equal(3, len(payload))
	s.Require().NotNil(payload["alice1"])
	s.Require().NotNil(payload["alice3"])
	s.Require().NotNil(payload["alice4"])

	// We disable the last installation
	err = s.services[aliceUser].services[0].DisableInstallation(&aliceKey.PublicKey, "alice4")
	s.Require().NoError(err)

	// We check the bundle is updated
	aliceBundle, err = alice1.GetBundle(aliceKey)
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
	msg, err = bob1.BuildDirectMessage(bobKey, &aliceKey.PublicKey, []byte("test"))
	s.Require().NoError(err)
	payload = msg.Message.GetDirectMessage()
	s.Require().Equal(3, len(payload))
	s.Require().NotNil(payload["alice1"])
	s.Require().NotNil(payload["alice2"])
	s.Require().NotNil(payload["alice3"])
}
