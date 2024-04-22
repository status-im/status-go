package pairing

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"gopkg.in/go-playground/validator.v9"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}

type ConfigTestSuite struct {
	suite.Suite
	validate *validator.Validate
}

func (s *ConfigTestSuite) SetupTest() {
	var err error
	s.validate, err = newValidate()
	require.NoError(s.T(), err, "newValidate should not return error")
}

func (s *ConfigTestSuite) TestValidationKeystorePath() {
	s.T().Run("Valid keystore path with keyUID", func(t *testing.T) {
		sc := &SenderConfig{
			KeystorePath: "some/path/0x130cc0ebdaecd220c1d6dea0ef01d575ef5364506785745049eb98ddf49cb54e",
			DeviceType:   "phone",
			KeyUID:       "0x130cc0ebdaecd220c1d6dea0ef01d575ef5364506785745049eb98ddf49cb54e",
			Password:     "password",
		}

		assert.NoError(t, s.validate.Struct(sc), "SenderConfig validation should pass")
	})

	s.T().Run("Invalid keystore path without keyUID", func(t *testing.T) {
		sc := &SenderConfig{
			KeystorePath: "some/path/",
			DeviceType:   "phone",
			KeyUID:       "0x130cc0ebdaecd220c1d6dea0ef01d575ef5364506785745049eb98ddf49cb54e",
			Password:     "password",
		}

		assert.Error(t, s.validate.Struct(sc), "SenderConfig validation should fail")
	})
}

func (s *ConfigTestSuite) TestValidationKeyUID() {
	s.T().Run("Valid keyUID", func(t *testing.T) {
		sc := &SenderConfig{
			KeystorePath: "some/path/0x130cc0ebdaecd220c1d6dea0ef01d575ef5364506785745049eb98ddf49cb54e",
			DeviceType:   "phone",
			KeyUID:       "0x130cc0ebdaecd220c1d6dea0ef01d575ef5364506785745049eb98ddf49cb54e",
			Password:     "password",
		}

		assert.NoError(t, s.validate.Struct(sc), "SenderConfig validation should pass")
	})

	s.T().Run("Invalid keyUID", func(t *testing.T) {
		sc := &SenderConfig{
			KeystorePath: "some/path/0x130cc0ebdaecd220c1d6dea0ef01d575ef5364506785745049eb98ddf49cb54e",
			DeviceType:   "phone",
			KeyUID:       "0x130cc0ebdaecd220c1d6dea0ef01d575ef5364506785745049eb98ddf49cb54",
			Password:     "password",
		}

		assert.Error(t, s.validate.Struct(sc), "SenderConfig validation should fail")
	})
}

func (s *ConfigTestSuite) TestValidationNotEndKeyUID() {
	nodeConfig, err := nodeConfigForLocalPairSync(uuid.New().String(), "", "/dummy/path")
	nodeConfig.RootDataDir = "/tmp"
	require.NoError(s.T(), err, "nodeConfigForLocalPairSync should not return error")
	s.T().Run("Valid keystore path without keyUID", func(t *testing.T) {
		r := &ReceiverConfig{
			NodeConfig:            nodeConfig,
			KeystorePath:          "some/path/",
			DeviceType:            "phone",
			KDFIterations:         1,
			SettingCurrentNetwork: "mainnet",
		}
		assert.NoError(t, setDefaultNodeConfig(r.NodeConfig))
		assert.NoError(t, validateAndVerifyNodeConfig(r, r), "ReceiverConfig validation should pass")
	})

	s.T().Run("Invalid keystore path with keyUID", func(t *testing.T) {
		r := &ReceiverConfig{
			NodeConfig:            nodeConfig,
			KeystorePath:          "some/path/0x130cc0ebdaecd220c1d6dea0ef01d575ef5364506785745049eb98ddf49cb54e",
			DeviceType:            "phone",
			KDFIterations:         1,
			SettingCurrentNetwork: "mainnet",
		}
		assert.NoError(t, setDefaultNodeConfig(r.NodeConfig))
		assert.Error(t, validateAndVerifyNodeConfig(r, r), "ReceiverConfig validation should fail")
	})
}
