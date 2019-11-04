module github.com/status-im/status-protocol-go

go 1.13

require (
	github.com/cenkalti/backoff/v3 v3.0.0
	github.com/ethereum/go-ethereum v1.9.5
	github.com/golang/protobuf v1.3.2
	github.com/google/uuid v1.1.1
	github.com/jinzhu/copier v0.0.0-20190625015134-976e0346caa8
	github.com/lucasb-eyer/go-colorful v1.0.2
	github.com/mutecomm/go-sqlcipher v0.0.0-20190227152316-55dbde17881f
	github.com/pkg/errors v0.8.1
	github.com/russolsen/transit v0.0.0-20180705123435-0794b4c4505a
	github.com/status-im/doubleratchet v3.0.0+incompatible
	github.com/status-im/migrate/v4 v4.6.2-status.2
	github.com/status-im/status-go v0.34.0-beta.3 // indirect
	github.com/status-im/whisper v1.5.1
	github.com/stretchr/testify v1.4.0
	github.com/vacp2p/mvds v0.0.23
	github.com/wealdtech/go-ens/v3 v3.0.7
	go.uber.org/zap v1.10.0
	golang.org/x/crypto v0.0.0-20191001141032-4663e185863a
)

replace github.com/ethereum/go-ethereum v1.9.5 => github.com/status-im/go-ethereum v1.9.5-status.4

replace github.com/NaySoftware/go-fcm => github.com/status-im/go-fcm v1.0.0-status
