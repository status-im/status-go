module github.com/status-im/status-go

go 1.13

replace github.com/ethereum/go-ethereum v1.9.5 => github.com/status-im/go-ethereum v1.9.5-status.4

replace github.com/NaySoftware/go-fcm => github.com/status-im/go-fcm v1.0.0-status

replace github.com/Sirupsen/logrus v1.4.2 => github.com/sirupsen/logrus v1.4.2

replace github.com/docker/docker => github.com/docker/engine v1.4.2-0.20190717161051-705d9623b7c1

require (
	github.com/NaySoftware/go-fcm v0.0.0-00010101000000-000000000000
	github.com/beevik/ntp v0.2.0
	github.com/btcsuite/btcd v0.0.0-20191011042131-c3151ef50de9
	github.com/btcsuite/btcutil v0.0.0-20190425235716-9e5f4b9a998d
	github.com/ethereum/go-ethereum v1.9.5
	github.com/golang/mock v1.3.1
	github.com/lib/pq v1.2.0
	github.com/libp2p/go-libp2p-core v0.2.3
	github.com/multiformats/go-multiaddr v0.1.1
	github.com/mutecomm/go-sqlcipher v0.0.0-20190227152316-55dbde17881f
	github.com/okzk/sdnotify v0.0.0-20180710141335-d9becc38acbd
	github.com/pborman/uuid v1.2.0
	github.com/prometheus/client_golang v1.2.1
	github.com/russolsen/transit v0.0.0-20180705123435-0794b4c4505a
	github.com/status-im/migrate/v4 v4.6.2-status.2
	github.com/status-im/rendezvous v1.3.0
	github.com/status-im/status-protocol-go v0.4.4-0.20191102185313-c99310ece510
	github.com/status-im/whisper v1.5.2
	github.com/stretchr/testify v1.4.0
	github.com/syndtr/goleveldb v1.0.0
	go.uber.org/zap v1.10.0
	golang.org/x/crypto v0.0.0-20191001141032-4663e185863a
	golang.org/x/text v0.3.2
	gopkg.in/go-playground/validator.v9 v9.29.1
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
)
