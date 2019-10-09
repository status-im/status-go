module github.com/status-im/status-protocol-go

go 1.12

require (
	github.com/aristanetworks/goarista v0.0.0-20190704150520-f44d68189fd7 // indirect
	github.com/cenkalti/backoff/v3 v3.0.0
	github.com/ethereum/go-ethereum v1.9.5
	github.com/golang/protobuf v1.3.2
	github.com/google/uuid v1.1.1
	github.com/jinzhu/copier v0.0.0-20190625015134-976e0346caa8
	github.com/lucasb-eyer/go-colorful v1.0.2
	github.com/mutecomm/go-sqlcipher v0.0.0-20190227152316-55dbde17881f
	github.com/pkg/errors v0.8.1
	github.com/russolsen/ohyeah v0.0.0-20160324131710-f4938c005315 // indirect
	github.com/russolsen/same v0.0.0-20160222130632-f089df61f51d // indirect
	github.com/russolsen/transit v0.0.0-20180705123435-0794b4c4505a
	github.com/status-im/doubleratchet v2.0.0+incompatible
	github.com/status-im/migrate/v4 v4.0.0-20190821140204-a9d340ec8fb76af4afda06acf01740d45d2661ed
	github.com/status-im/whisper v1.5.1
	github.com/stretchr/testify v1.3.1-0.20190712000136-221dbe5ed467
	github.com/vacp2p/mvds v0.0.21
	go.uber.org/zap v1.10.0
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4
	golang.org/x/net v0.0.0-20190628185345-da137c7871d7 // indirect
	golang.org/x/sys v0.0.0-20190801041406-cbf593c0f2f3 // indirect
)

replace github.com/ethereum/go-ethereum v1.9.5 => github.com/status-im/go-ethereum v1.9.5-status.4
