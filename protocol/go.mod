module github.com/status-im/status-go/protocol

go 1.13

replace github.com/ethereum/go-ethereum v1.9.5 => github.com/status-im/go-ethereum v1.9.5-status.6

replace github.com/gomarkdown/markdown => github.com/status-im/markdown v0.0.0-20191113114344-af599402d015

replace github.com/status-im/status-go/eth-node => ../eth-node

require (
	github.com/cenkalti/backoff/v3 v3.1.1
	github.com/golang/protobuf v1.3.2
	github.com/gomarkdown/markdown v0.0.0-20191113114344-af599402d015
	github.com/google/uuid v1.1.1
	github.com/jinzhu/copier v0.0.0-20190625015134-976e0346caa8
	github.com/lucasb-eyer/go-colorful v1.0.2
	github.com/mutecomm/go-sqlcipher v0.0.0-20190227152316-55dbde17881f
	github.com/pkg/errors v0.8.1
	github.com/russolsen/ohyeah v0.0.0-20160324131710-f4938c005315 // indirect
	github.com/russolsen/same v0.0.0-20160222130632-f089df61f51d // indirect
	github.com/russolsen/transit v0.0.0-20180705123435-0794b4c4505a
	github.com/status-im/doubleratchet v3.0.0+incompatible
	github.com/status-im/migrate/v4 v4.6.2-status.2
	github.com/status-im/status-go/eth-node v0.0.0-20191120100713-5053b0b6835b
	github.com/status-im/whisper v1.5.2
	github.com/stretchr/testify v1.4.0
	github.com/vacp2p/mvds v0.0.23
	go.uber.org/zap v1.13.0
)
