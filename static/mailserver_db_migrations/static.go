// Package static embeds static (JS, HTML) resources right into the binaries
package static

//go:generate go-bindata -pkg migrations -o ../../mailserver/migrations/bindata.go .
