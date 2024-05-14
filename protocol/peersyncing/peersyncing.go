package peersyncing

import "github.com/status-im/status-go/eth-node/types"

type PeerSyncing struct {
	persistence SyncMessagePersistence
	config      Config
}

func New(config Config) *PeerSyncing {
	syncMessagePersistence := config.SyncMessagePersistence
	if syncMessagePersistence == nil {
		syncMessagePersistence = NewSyncMessageSQLitePersistence(config.Database)
	}

	return &PeerSyncing{
		config:      config,
		persistence: syncMessagePersistence,
	}
}

func (p *PeerSyncing) Add(message SyncMessage) error {
	return p.persistence.Add(message)
}

func (p *PeerSyncing) AvailableMessages() ([]SyncMessage, error) {
	return p.persistence.All()
}

func (p *PeerSyncing) AvailableMessagesMapByChatIDs(groupIDs [][]byte, limit int) (map[string][][]byte, error) {
	availableMessages, err := p.persistence.ByChatIDs(groupIDs, limit)
	if err != nil {
		return nil, err
	}
	availableMessagesMap := make(map[string][][]byte)
	for _, m := range availableMessages {
		chatID := types.Bytes2Hex(m.ChatID)
		availableMessagesMap[chatID] = append(availableMessagesMap[chatID], m.ID)
	}
	return availableMessagesMap, err
}

func (p *PeerSyncing) MessagesByIDs(messageIDs [][]byte) ([]SyncMessage, error) {
	return p.persistence.ByMessageIDs(messageIDs)
}

func (p *PeerSyncing) OnOffer(messages []SyncMessage) ([]SyncMessage, error) {
	return p.persistence.Complement(messages)
}
