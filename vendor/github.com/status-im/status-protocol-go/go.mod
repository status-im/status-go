module github.com/status-im/status-protocol-go

go 1.12

require (
	github.com/aristanetworks/goarista v0.0.0-20190704150520-f44d68189fd7 // indirect
	github.com/deckarep/golang-set v1.7.1 // indirect
	github.com/ethereum/go-ethereum v1.8.27
	github.com/golang/protobuf v1.3.2
	github.com/jinzhu/copier v0.0.0-20190625015134-976e0346caa8
	github.com/mutecomm/go-sqlcipher v0.0.0-20190227152316-55dbde17881f
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/onsi/gomega v1.5.0 // indirect
	github.com/pkg/errors v0.8.1
	github.com/rs/cors v1.6.0 // indirect
	github.com/russolsen/ohyeah v0.0.0-20160324131710-f4938c005315 // indirect
	github.com/russolsen/same v0.0.0-20160222130632-f089df61f51d // indirect
	github.com/russolsen/transit v0.0.0-20180705123435-0794b4c4505a
	github.com/status-im/doubleratchet v2.0.0+incompatible
	github.com/status-im/migrate/v4 v4.0.0-20190821140204-a9d340ec8fb76af4afda06acf01740d45d2661ed
	github.com/status-im/whisper v1.4.14
	github.com/stretchr/testify v1.3.1-0.20190712000136-221dbe5ed467
	github.com/vacp2p/mvds v0.0.21
	go.uber.org/zap v1.10.0
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4
	golang.org/x/net v0.0.0-20190628185345-da137c7871d7 // indirect
)

replace github.com/ethereum/go-ethereum v1.8.27 => github.com/status-im/go-ethereum v1.8.27-status.4
