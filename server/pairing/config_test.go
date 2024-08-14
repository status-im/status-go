package pairing

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/suite"
	"gopkg.in/go-playground/validator.v9"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/protocol/requests"
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
	keyUIDPattern := regexp.MustCompile(`^0x[0-9a-fA-F]{64}$`)

	r := &ReceiverConfig{
		CreateAccount: &requests.CreateAccount{
			RootDataDir:   "/tmp",
			KdfIterations: 1,
			DeviceName:    "device-1",
		},
	}
	keystorePath := r.AbsoluteKeystorePath()
	s.Require().True(len(keystorePath) <= 66 || !keyUIDPattern.MatchString(keystorePath[len(keystorePath)-66:]))

	s.Require().NoError(validateReceiverConfig(r, r), "ReceiverConfig validation should pass")
}
