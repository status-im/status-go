package types

// ReceivedMessage is the neutral, transport-agnostic representation of a message
// received from the messaging backend. It mirrors the logos-delivery Messaging
// API MessageReceived event so the same seam can be satisfied by logos-delivery
// once the go-waku adapter is retired (pm#380); the go-waku WakuMessage is an
// internal detail and must not leak through this boundary.
//
// TODO(status-im/status-go#7522): shrink to the pure send-symmetric shape
// {Hash, ContentTopic, Payload, Ephemeral, Meta} once the v1->v0 cutover
// (pm#420) removes the decode's dependency on Version and Timestamp is sourced
// elsewhere. They are carried transitionally because the rfc26 decode still
// needs Version (for the version==0 passthrough) and the processor needs
// Timestamp (Message.Timestamp / "sent" time).
type ReceivedMessage struct {
	Hash         []byte
	ContentTopic string
	Payload      []byte
	Ephemeral    bool
	Meta         []byte

	// Transitional (see TODO above).
	Version   uint32
	Timestamp int64
}
