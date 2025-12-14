// Package static embeds static (JS, HTML) resources right into the binaries
package static

//go:generate go tool go-bindata -modtime=1700000000 -pkg static -o bindata.go ./keys
