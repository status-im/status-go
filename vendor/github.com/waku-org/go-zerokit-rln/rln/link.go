package rln

/*
#cgo LDFLAGS:-lrln -ldl -lm
#cgo linux,arm LDFLAGS:-L${SRCDIR}/../libs/armv7-linux-androideabi
#cgo linux,arm64 LDFLAGS:-L${SRCDIR}/../libs/aarch64-unknown-linux-gnu
#cgo linux,amd64,musl,!android LDFLAGS:-L${SRCDIR}/../libs/x86_64-unknown-linux-musl
#cgo linux,amd64,!musl,!android LDFLAGS:-L${SRCDIR}/../libs/x86_64-unknown-linux-gnu
#cgo linux,386 LDFLAGS:-L${SRCDIR}/../libs/i686-unknown-linux-gnu
#cgo windows,386 LDFLAGS:-L${SRCDIR}/../libs/i686-pc-windows-gnu -lrln -lm -lws2_32 -luserenv
#cgo windows,amd64 LDFLAGS:-L${SRCDIR}/../libs/x86_64-pc-windows-gnu -lrln -lm -lws2_32 -luserenv
#cgo darwin,386,!ios LDFLAGS:-L${SRCDIR}/../libs/i686-apple-darwin
#cgo darwin,arm64,!ios LDFLAGS:-L${SRCDIR}/../libs/aarch64-apple-darwin
#cgo darwin,amd64,!ios LDFLAGS:-L${SRCDIR}/../libs/x86_64-apple-darwin
*/
import "C"
