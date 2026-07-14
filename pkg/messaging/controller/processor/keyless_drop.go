package processor

// shouldDropUndecryptableHashRatchetMessage reports whether an incoming
// hash-ratchet message that could not be decrypted (ErrHashRatchetGroupIDNotFound)
// should be dropped instead of parked for retroactive replay.
//
// It drops only when the feature is enabled AND the node holds no key material at
// all for the message's group — i.e. a keyless spectator receiving a community
// channel's encrypted data plane (issue #21470). A member missing only this
// rotation's key still holds other keys for the group, so the message is kept so
// it can be replayed once the newer key arrives.
//
// Key-exchange messages (HR header SeqNo == 0) decrypt successfully and never
// reach this branch, so a drop can never discard key-distribution traffic.
func shouldDropUndecryptableHashRatchetMessage(enabled, holdsKeys bool) bool {
	return enabled && !holdsKeys
}
