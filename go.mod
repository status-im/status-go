module github.com/status-im/status-go

go 1.13

replace github.com/ethereum/go-ethereum v1.9.5 => github.com/status-im/go-ethereum v1.9.5-status.6

replace github.com/Sirupsen/logrus v1.4.2 => github.com/sirupsen/logrus v1.4.2

replace github.com/docker/docker => github.com/docker/engine v1.4.2-0.20190717161051-705d9623b7c1

replace github.com/gomarkdown/markdown => github.com/status-im/markdown v0.0.0-20191113114344-af599402d015

replace github.com/status-im/status-go/protocol => ./protocol

replace github.com/status-im/status-go/extkeys => ./extkeys

replace github.com/status-im/status-go/eth-node => ./eth-node

require (
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20190717042225-c3de453c63f4 // indirect
	github.com/beevik/ntp v0.2.0
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.0 // indirect
	github.com/elastic/gosigar v0.10.5 // indirect
	github.com/ethereum/go-ethereum v1.9.5
	github.com/go-kit/kit v0.9.0 // indirect
	github.com/go-playground/locales v0.12.1 // indirect
	github.com/go-playground/universal-translator v0.16.0 // indirect
	github.com/golang/mock v1.3.1
	github.com/json-iterator/go v1.1.7 // indirect
	github.com/kevinburke/go-bindata v3.13.0+incompatible // indirect
	github.com/leodido/go-urn v1.1.0 // indirect
	github.com/lib/pq v1.2.0
	github.com/libp2p/go-libp2p v0.4.0 // indirect
	github.com/libp2p/go-libp2p-core v0.2.3
	github.com/multiformats/go-multiaddr v0.1.1
	github.com/mutecomm/go-sqlcipher v0.0.0-20190227152316-55dbde17881f
	github.com/okzk/sdnotify v0.0.0-20180710141335-d9becc38acbd
	github.com/pborman/uuid v1.2.0
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v1.0.0
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4 // indirect
	github.com/prometheus/common v0.6.0 // indirect
	github.com/prometheus/procfs v0.0.5 // indirect
	github.com/russolsen/transit v0.0.0-20180705123435-0794b4c4505a
	github.com/sirupsen/logrus v1.4.2 // indirect
	github.com/status-im/migrate/v4 v4.6.2-status.2
	github.com/status-im/rendezvous v1.3.0
	github.com/status-im/status-go/eth-node v0.0.0-20191126161717-86bc127b3d0a
	github.com/status-im/status-go/extkeys v1.0.0
	github.com/status-im/status-go/protocol v0.0.0-00010101000000-000000000000
	github.com/status-im/tcp-shaker v0.0.0-20191114194237-215893130501
	github.com/status-im/whisper v1.6.1
	github.com/stretchr/testify v1.4.0
	github.com/syndtr/goleveldb v1.0.0
	go.uber.org/zap v1.13.0
	golang.org/x/crypto v0.0.0-20191119213627-4f8c1d86b1ba
	golang.org/x/net v0.0.0-20190930134127-c5a3c61f89f3 // indirect
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e // indirect
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/go-playground/validator.v9 v9.29.1
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
)
