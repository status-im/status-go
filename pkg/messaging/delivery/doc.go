// Package delivery hosts the logos-delivery backend for the messaging layer
// (pm#380). It wraps the Messaging API exposed by
// logos-delivery-go-bindings/pkg/messaging, which in turn drives
// liblogosdelivery over cgo.
//
// At this point the package only proves that status-go links against
// liblogosdelivery; the adapter that satisfies transport.MessagingAPI lands on
// top of it.
//
// Building this package requires liblogosdelivery. The Nix dev shell provides
// it (LOGOS_DELIVERY_LIB_DIR / LOGOS_DELIVERY_INC_DIR, turned into CGO flags by
// the Makefile); outside Nix, set those two variables to a logos-delivery
// build.
package delivery
