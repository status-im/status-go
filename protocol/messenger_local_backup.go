package protocol

import (
	"errors"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/signal"
)

func (m *Messenger) ExportBackup() ([]byte, error) {
	backup := &protobuf.MessengerLocalBackup{}

	clock, _ := m.getLastClockWithRelatedChat()
	contactsToBackup := m.backupContacts(m.ctx)
	communitiesToBackup, err := m.backupCommunities(m.ctx, clock)
	if err != nil {
		return nil, err
	}
	chatsToBackup := m.backupChats(m.ctx, clock)
	profileToBackup, err := m.backupProfile(m.ctx, clock)
	if err != nil {
		return nil, err
	}

	for _, d := range contactsToBackup {
		backup.Contacts = append(backup.Contacts, d.Contacts...)
	}
	for _, d := range communitiesToBackup {
		backup.Communities = append(backup.Communities, d.Communities...)
	}
	backup.Profile = profileToBackup.Profile
	for _, d := range chatsToBackup {
		backup.Chats = append(backup.Chats, d.Chats...)
	}
	return proto.Marshal(backup)
}

func (m *Messenger) ImportBackup(data []byte) error {
	var backup protobuf.MessengerLocalBackup
	err := proto.Unmarshal(data, &backup)
	if err != nil {
		return err
	}

	state := ReceivedMessageState{
		Response: &MessengerResponse{},
		AllChats: &chatMap{},
		AllContacts: &contactMap{
			me: m.selfContact,
		},
		Timesource:            m.getTimesource(),
		ModifiedContacts:      &stringBoolMap{},
		ModifiedInstallations: &stringBoolMap{},
	}
	errs := m.handleLocalBackup(
		&state,
		&backup,
	)
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	response, err := m.saveDataAndPrepareResponse(&state)

	signal.SendNewMessages(response)

	return err
}
