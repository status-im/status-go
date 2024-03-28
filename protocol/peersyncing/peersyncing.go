package peersyncing

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

func (p *PeerSyncing) AvailableMessagesByChatID(groupID []byte, limit int) ([]SyncMessage, error) {
	return p.persistence.ByChatID(groupID, limit)
}

func (p *PeerSyncing) AvailableMessagesByChatIDs(groupIDs [][]byte, limit int) ([]SyncMessage, error) {
	return p.persistence.ByChatIDs(groupIDs, limit)
}

func (p *PeerSyncing) MessagesByIDs(messageIDs [][]byte) ([]SyncMessage, error) {
	return p.persistence.ByMessageIDs(messageIDs)
}

func (p *PeerSyncing) OnOffer(messages []SyncMessage) ([]SyncMessage, error) {
	return p.persistence.Complement(messages)
}
