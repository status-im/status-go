// Package delivery is the messaging backend over the logos-delivery Messaging
// API. Adapter satisfies the same seam as the go-waku backend (types.Waku and
// transport.MessagingAPI), driving a Client.
//
// The package itself is pure Go. The Client backed by liblogosdelivery, through
// logos-delivery-go-bindings, is only built with the logos_delivery build tag,
// so default builds neither link the library nor need it installed.
package delivery
