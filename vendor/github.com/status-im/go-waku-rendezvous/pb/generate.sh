#!/bin/bash

# rendezvous.proto
protoc --gofast_out=. --proto_path=$(go list -f '{{ .Dir }}' -m github.com/libp2p/go-libp2p-core) --proto_path=. rendezvous.proto
sed -i "s/record\/pb/github.com\/libp2p\/go-libp2p-core\/record\/pb/" rendezvous.pb.go
