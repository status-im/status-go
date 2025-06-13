package protocol

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/status-im/status-go/params"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/requests"
)

func TestMessengerLocalBackupSuite(t *testing.T) {
	suite.Run(t, new(MessengerBackupSuite))
}

type MessengerLocalBackupSuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerBackupSuite) TestBackupContactsLocally() {
	backupOptions := []Option{
		WithLocalBackup(&params.BackupConfig{
			DataDir: filepath.Join(s.tmpdir, params.BackupsRelativePath),
		}),
	}

	// Create bob1
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	bob1, err := newMessengerWithKey(s.shh, privateKey, s.logger, backupOptions)
	s.Require().NoError(err)
	defer TearDownMessenger(&s.Suite, bob1)

	// Create bob2
	bob2, err := newMessengerWithKey(s.shh, bob1.identity, s.logger, backupOptions)
	s.Require().NoError(err)
	defer TearDownMessenger(&s.Suite, bob2)

	// Make sure there is no backup at first
	backupFile := filepath.Join(bob1.config.backupConfig.DataDir, "user_data.bkp")
	err = os.RemoveAll(backupFile)
	s.Require().NoError(err)

	// Create 2 contacts
	contact1Key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	contactID1 := types.EncodeHex(crypto.FromECDSAPub(&contact1Key.PublicKey))

	_, err = bob1.AddContact(context.Background(), &requests.AddContact{ID: contactID1})
	s.Require().NoError(err)

	contact2Key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	contactID2 := types.EncodeHex(crypto.FromECDSAPub(&contact2Key.PublicKey))

	_, err = bob1.AddContact(context.Background(), &requests.AddContact{ID: contactID2})
	s.Require().NoError(err)

	s.Require().Len(bob1.Contacts(), 2)

	actualContacts := bob1.Contacts()
	if actualContacts[0].ID == contactID1 {
		s.Require().Equal(actualContacts[0].ID, contactID1)
		s.Require().Equal(actualContacts[1].ID, contactID2)
	} else {
		s.Require().Equal(actualContacts[0].ID, contactID2)
		s.Require().Equal(actualContacts[1].ID, contactID1)
	}

	s.Require().Equal(ContactRequestStateSent, actualContacts[0].ContactRequestLocalState)
	s.Require().Equal(ContactRequestStateSent, actualContacts[1].ContactRequestLocalState)
	s.Require().True(actualContacts[0].added())
	s.Require().True(actualContacts[1].added())

	// Backup
	err = bob1.BackupDataLocally(context.Background())
	s.Require().NoError(err)

	// Safety check
	s.Require().Len(bob2.Contacts(), 0)

	// Import the backup file and process it
	response, err := bob2.importLocalBackupFile(backupFile)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Contacts, 2)
	s.Require().Len(bob2.Contacts(), 2)
}
