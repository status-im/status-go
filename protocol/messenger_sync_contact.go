package protocol

import (
	multiaccountscommon "github.com/status-im/status-go/multiaccounts/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

func (m *Messenger) buildSyncContactMessage(contact *Contact) *protobuf.SyncInstallationContactV2 {
	var ensName string
	if contact.ENSVerified {
		ensName = contact.EnsName
	}

	var customizationColor uint32
	if len(contact.CustomizationColor) != 0 {
		customizationColor = multiaccountscommon.ColorToIDFallbackToBlue(contact.CustomizationColor)
	}

	oneToOneChat, ok := m.allChats.Load(contact.ID)
	muted := false
	if ok {
		muted = oneToOneChat.Muted
	}

	return &protobuf.SyncInstallationContactV2{
		LastUpdatedLocally:        contact.LastUpdatedLocally,
		LastUpdated:               contact.LastUpdated,
		Id:                        contact.ID,
		DisplayName:               contact.DisplayName,
		EnsName:                   ensName,
		CustomizationColor:        customizationColor,
		LocalNickname:             contact.LocalNickname,
		Added:                     contact.added(),
		Blocked:                   contact.Blocked,
		Muted:                     muted,
		HasAddedUs:                contact.hasAddedUs(),
		Removed:                   contact.Removed,
		ContactRequestLocalState:  int64(contact.ContactRequestLocalState),
		ContactRequestRemoteState: int64(contact.ContactRequestRemoteState),
		ContactRequestRemoteClock: int64(contact.ContactRequestRemoteClock),
		ContactRequestLocalClock:  int64(contact.ContactRequestLocalClock),
		VerificationStatus:        int64(contact.VerificationStatus),
		TrustStatus:               int64(contact.TrustStatus),
	}
}
