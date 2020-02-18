// Bridge bridges Whisper and Waku subprotocols.
// This is possible because both use the same envelope format.
// What's more, both envelope formats are identical structs,
// that is having the same ordered fields.

package bridge
