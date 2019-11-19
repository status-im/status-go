module github.com/status-im/status-protocol-go

go 1.13

require (
	github.com/aristanetworks/goarista v0.0.0-20190704150520-f44d68189fd7 // indirect
	github.com/cenkalti/backoff/v3 v3.0.0
	github.com/ethereum/go-ethereum v1.9.5
	github.com/golang/protobuf v1.3.2
	github.com/gomarkdown/markdown v0.0.0-20191113114344-af599402d015
	github.com/google/uuid v1.1.1
	github.com/jinzhu/copier v0.0.0-20190625015134-976e0346caa8
	github.com/karalabe/usb v0.0.0-20190919080040-51dc0efba356 // indirect
	github.com/lucasb-eyer/go-colorful v1.0.2
	github.com/mattn/go-pointer v0.0.0-20190911064623-a0a44394634f
	github.com/minio/sha256-simd v0.1.1 // indirect
	github.com/multiformats/go-multihash v0.0.8 // indirect
	github.com/mutecomm/go-sqlcipher v0.0.0-20190227152316-55dbde17881f
	github.com/pkg/errors v0.8.1
	github.com/russolsen/ohyeah v0.0.0-20160324131710-f4938c005315 // indirect
	github.com/russolsen/same v0.0.0-20160222130632-f089df61f51d // indirect
	github.com/russolsen/transit v0.0.0-20180705123435-0794b4c4505a
	github.com/status-im/doubleratchet v3.0.0+incompatible
	github.com/status-im/migrate/v4 v4.6.2-status.2
	github.com/status-im/whisper v1.5.2
	github.com/stretchr/testify v1.4.0
	github.com/vacp2p/mvds v0.0.23
	github.com/wealdtech/go-ens/v3 v3.0.7
	go.uber.org/zap v1.10.0
	golang.org/x/crypto v0.0.0-20191029031824-8986dd9e96cf
	golang.org/x/net v0.0.0-20190930134127-c5a3c61f89f3 // indirect
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e // indirect
	golang.org/x/sys v0.0.0-20190927073244-c990c680b611 // indirect
)

replace github.com/ethereum/go-ethereum v1.9.5 => github.com/status-im/go-ethereum v1.9.5-status.5

replace github.com/gomarkdown/markdown => github.com/status-im/markdown v0.0.0-20191113114344-af599402d015
