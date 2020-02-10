module github.com/status-im/status-go

go 1.13

replace github.com/ethereum/go-ethereum v1.9.5 => github.com/status-im/go-ethereum v1.9.5-status.7

replace github.com/Sirupsen/logrus v1.4.2 => github.com/sirupsen/logrus v1.4.2

replace github.com/docker/docker => github.com/docker/engine v1.4.2-0.20190717161051-705d9623b7c1

replace github.com/gomarkdown/markdown v0.0.0-20191209105822-e3ba6c6109ba => github.com/status-im/markdown v0.0.0-20191209105822-e3ba6c6109ba

replace github.com/status-im/status-go/protocol => ./protocol

replace github.com/status-im/status-go/extkeys => ./extkeys

replace github.com/status-im/status-go/eth-node => ./eth-node

replace github.com/status-im/status-go/whisper/v6 => ./whisper

replace github.com/status-im/status-go/waku => ./waku

require (
	github.com/beevik/ntp v0.2.0
	github.com/ethereum/go-ethereum v1.9.5
	github.com/go-playground/universal-translator v0.17.0 // indirect
	github.com/golang/mock v1.3.1
	github.com/leodido/go-urn v1.2.0 // indirect
	github.com/lib/pq v1.2.0
	github.com/libp2p/go-libp2p v0.4.2 // indirect
	github.com/libp2p/go-libp2p-core v0.2.4
	github.com/multiformats/go-multiaddr v0.1.1
	github.com/mutecomm/go-sqlcipher v0.0.0-20190227152316-55dbde17881f
	github.com/okzk/sdnotify v0.0.0-20180710141335-d9becc38acbd
	github.com/pborman/uuid v1.2.0
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v1.2.1
	github.com/russolsen/ohyeah v0.0.0-20160324131710-f4938c005315 // indirect
	github.com/russolsen/same v0.0.0-20160222130632-f089df61f51d // indirect
	github.com/russolsen/transit v0.0.0-20180705123435-0794b4c4505a
	github.com/status-im/migrate/v4 v4.6.2-status.2
	github.com/status-im/rendezvous v1.3.0
	github.com/status-im/status-go/eth-node v1.1.0
	github.com/status-im/status-go/extkeys v1.1.0
	github.com/status-im/status-go/protocol v1.1.0
	github.com/status-im/status-go/waku v1.2.0
	github.com/status-im/status-go/whisper/v6 v6.1.0
	github.com/status-im/tcp-shaker v0.0.0-20191114194237-215893130501
	github.com/stretchr/testify v1.4.0
	github.com/syndtr/goleveldb v1.0.0
	go.uber.org/zap v1.13.0
	golang.org/x/crypto v0.0.0-20191122220453-ac88ee75c92c
	golang.org/x/tools v0.0.0-20200116062425-473961ec044c // indirect
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/go-playground/validator.v9 v9.31.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
)
