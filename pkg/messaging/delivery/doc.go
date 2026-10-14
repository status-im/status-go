// Package delivery is the messaging backend over the logos-delivery Messaging
// API. Adapter satisfies the same seam as the go-waku backend (types.Waku and
// transport.MessagingAPI), driving a Client.
//
// NewClient creates the Client backed by liblogosdelivery, through
// logos-delivery-go-bindings.
package delivery
