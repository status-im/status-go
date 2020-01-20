module github.com/status-im/status-go/protocol

go 1.13

replace github.com/ethereum/go-ethereum v1.9.5 => github.com/status-im/go-ethereum v1.9.5-status.7

replace github.com/gomarkdown/markdown v0.0.0-20191209105822-e3ba6c6109ba => github.com/status-im/markdown v0.0.0-20191209105822-e3ba6c6109ba

replace github.com/status-im/status-go/eth-node => ../eth-node

replace github.com/status-im/status-go/whisper/v6 => ../whisper

require (
	github.com/cenkalti/backoff/v3 v3.2.2
	github.com/ethereum/go-ethereum v1.9.5
	github.com/golang/protobuf v1.3.2
	github.com/gomarkdown/markdown v0.0.0-20191209105822-e3ba6c6109ba
	github.com/google/uuid v1.1.1
	github.com/jinzhu/copier v0.0.0-20190625015134-976e0346caa8
	github.com/lucasb-eyer/go-colorful v1.0.2
	github.com/mutecomm/go-sqlcipher v0.0.0-20190227152316-55dbde17881f
	github.com/pkg/errors v0.8.1
	github.com/russolsen/transit v0.0.0-20180705123435-0794b4c4505a
	github.com/status-im/doubleratchet v3.0.0+incompatible
	github.com/status-im/migrate/v4 v4.6.2-status.2
	github.com/status-im/status-go/eth-node v1.1.0
	github.com/status-im/status-go/whisper/v6 v6.1.0
	github.com/stretchr/testify v1.4.0
	github.com/vacp2p/mvds v0.0.23
	go.uber.org/zap v1.13.0
)
