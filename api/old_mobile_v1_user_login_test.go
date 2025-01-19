package api

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/status-im/status-go/common"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/t/utils"
)

const (
	v1_10_keyUID              = "0x88f310d80e3d5821c00714c52bf4fae15f571ba5abae6d804b1e8a9723136a9c"
	v1_10_passwd              = "0x20756cad9b728c8225fd8cedb6badaf8731e174506950219ea657cd54f35f46c" // #nosec G101
	v1_10_BeforeUpgradeFolder = "../static/test-mobile-release-1.10.1/before-upgrade-to-v2.30.0"
	// This folder is used to test the case where the node config migration failed
	// and the user logs in again after the failure
	v1_10_AfterUpgradeFolder = "../static/test-mobile-release-1.10.1/after-upgrade-to-v2.30.0"
)

type OldMobileV1_10_UserLoginTest struct {
	suite.Suite
	tmpdir string
	logger *zap.Logger
}

func (s *OldMobileV1_10_UserLoginTest) SetupTest() {
	utils.Init()
	var err error
	s.logger, err = zap.NewDevelopment()
	s.Require().NoError(err)
}

func TestOldMobileV1_10_UserLogin(t *testing.T) {
	suite.Run(t, new(OldMobileV1_10_UserLoginTest))
}

func (s *OldMobileV1_10_UserLoginTest) TestLoginWithSuccessNodeConfigMigration() {
	s.tmpdir = s.T().TempDir()
	copyDir(v1_10_BeforeUpgradeFolder, s.tmpdir, s.T())

	b := NewGethStatusBackend(s.logger)
	b.UpdateRootDataDir(s.tmpdir)
	s.Require().NoError(b.OpenAccounts())
	loginRequest := &requests.Login{
		KeyUID:   v1_10_keyUID,
		Password: v1_10_passwd,
	}
	s.Require().NoError(b.LoginAccount(loginRequest))
	s.Require().NoError(b.Logout())
}

// without workaroundToFixBadMigration, this test would login fail
func (s *OldMobileV1_10_UserLoginTest) TestLoginWithFailNodeConfigMigration() {
	bkFunc := common.IsMobilePlatform
	common.IsMobilePlatform = func() bool {
		return true
	}
	defer func() {
		common.IsMobilePlatform = bkFunc
	}()

	s.tmpdir = s.T().TempDir()
	copyDir(v1_10_AfterUpgradeFolder, s.tmpdir, s.T())

	b := NewGethStatusBackend(s.logger)
	b.UpdateRootDataDir(s.tmpdir)
	s.Require().NoError(b.OpenAccounts())
	loginRequest := &requests.Login{
		KeyUID:   v1_10_keyUID,
		Password: v1_10_passwd,
	}
	err := b.LoginAccount(loginRequest)
	s.Require().NoError(err)
}
