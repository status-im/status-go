package cbridge

//go:generate sh -c "PATH=\"$(go list -m -f '{{.Dir}}')/_assets/scripts:$PATH\" protoc --go_out=. ./cbridge.proto ./gateway.proto ./query.proto"
