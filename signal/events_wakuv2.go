package signal

const (
	// EventPeerStats is sent when peer is added or removed.
	// it will be a map with capability=peer count k/v's.
	EventPeerStats = "wakuv2.peerstats"
)

// SendPeerStats sends discovery.summary signal.
func SendPeerStats(peerStats interface{}) {
	send(EventPeerStats, peerStats)
}
