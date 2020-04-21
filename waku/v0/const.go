package v0

// Waku protocol parameters
const (
	Version    = uint64(0) // Peer version number
	VersionStr = "0"       // The same, as a string
	Name       = "waku"    // Nickname of the protocol

	// Waku protocol message codes, according to https://github.com/vacp2p/specs/blob/master/specs/waku/waku-0.md
	StatusCode             = 0   // used in the handshake
	MessagesCode           = 1   // regular message
	StatusUpdateCode       = 22  // update of settings
	BatchAcknowledgedCode  = 11  // confirmation that batch of envelopes was received
	MessageResponseCode    = 12  // includes confirmation for delivery and information about errors
	P2PRequestCompleteCode = 125 // peer-to-peer message, used by Dapp protocol
	P2PRequestCode         = 126 // peer-to-peer message, used by Dapp protocol
	P2PMessageCode         = 127 // peer-to-peer message (to be consumed by the peer, but not forwarded any further)
	NumberOfMessageCodes   = 128
)
