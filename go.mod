module github.com/status-im/status-go

go 1.13

replace github.com/ethereum/go-ethereum v1.9.5 => github.com/status-im/go-ethereum v1.9.5-status.12

replace github.com/Sirupsen/logrus v1.4.2 => github.com/sirupsen/logrus v1.4.2

replace github.com/docker/docker => github.com/docker/engine v1.4.2-0.20190717161051-705d9623b7c1

replace github.com/nfnt/resize => github.com/status-im/resize v0.0.0-20201215164250-7c6d9f0d3088

require (
	github.com/PuerkitoBio/goquery v1.6.0 // indirect
	github.com/beevik/ntp v0.2.0
	github.com/cenkalti/backoff/v3 v3.2.2
	github.com/davecgh/go-spew v1.1.1
	github.com/deckarep/golang-set v1.7.1
	github.com/ethereum/go-ethereum v1.9.5
	github.com/gballet/go-libpcsclite v0.0.0-20191108122812-4678299bea08 // indirect
	github.com/go-playground/universal-translator v0.17.0 // indirect
	github.com/golang-migrate/migrate/v4 v4.8.0 // indirect
	github.com/golang/groupcache v0.0.0-20191027212112-611e8accdfc9 // indirect
	github.com/golang/mock v1.4.1
	github.com/golang/protobuf v1.3.4
	github.com/google/go-cmp v0.5.5 // indirect
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.7.3 // indirect
	github.com/jinzhu/copier v0.0.0-20190924061706-b57f9002281a
	github.com/karalabe/usb v0.0.0-20191104083709-911d15fe12a9 // indirect
	github.com/keighl/metabolize v0.0.0-20150915210303-97ab655d4034
	github.com/kilic/bls12-381 v0.0.0-20200607163746-32e1441c8a9f
	github.com/leodido/go-urn v1.2.0 // indirect
	github.com/lib/pq v1.9.0
	github.com/libp2p/go-libp2p v0.4.2 // indirect
	github.com/libp2p/go-libp2p-core v0.2.4
	github.com/lucasb-eyer/go-colorful v1.0.3
	github.com/mat/besticon v3.12.0+incompatible
	github.com/mattn/go-colorable v0.1.4 // indirect
	github.com/mattn/go-isatty v0.0.10 // indirect
	github.com/mattn/go-runewidth v0.0.6 // indirect
	github.com/mattn/go-sqlite3 v1.12.0 // indirect
	github.com/multiformats/go-multiaddr v0.1.1
	github.com/multiformats/go-multibase v0.0.1
	github.com/multiformats/go-varint v0.0.5
	github.com/mutecomm/go-sqlcipher v0.0.0-20190227152316-55dbde17881f
	github.com/nfnt/resize v0.0.0-00010101000000-000000000000
	github.com/okzk/sdnotify v0.0.0-20180710141335-d9becc38acbd
	github.com/olekukonko/tablewriter v0.0.2 // indirect
	github.com/oliamb/cutter v0.2.2
	github.com/onsi/ginkgo v1.10.3 // indirect
	github.com/onsi/gomega v1.7.1 // indirect
	github.com/pborman/uuid v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.5.0
	github.com/russolsen/ohyeah v0.0.0-20160324131710-f4938c005315 // indirect
	github.com/russolsen/same v0.0.0-20160222130632-f089df61f51d // indirect
	github.com/russolsen/transit v0.0.0-20180705123435-0794b4c4505a
	github.com/status-im/doubleratchet v3.0.0+incompatible
	github.com/status-im/keycard-go v0.0.0-20200107115650-f38e9a19958e // indirect
	github.com/status-im/markdown v0.0.0-20201022101546-c0cbdd5763bf
	github.com/status-im/migrate/v4 v4.6.2-status.2
	github.com/status-im/rendezvous v1.3.0
	github.com/status-im/status-go/extkeys v1.1.2
	github.com/status-im/tcp-shaker v0.0.0-20191114194237-215893130501
	github.com/stretchr/testify v1.5.1
	github.com/syndtr/goleveldb v1.0.0
	github.com/tsenart/tb v0.0.0-20181025101425-0d2499c8b6e9
	github.com/vacp2p/mvds v0.0.24-0.20201124060106-26d8e94130d8
	github.com/wealdtech/go-ens/v3 v3.3.0
	github.com/xeipuuv/gojsonschema v1.2.0
	go.uber.org/multierr v1.4.0 // indirect
	go.uber.org/zap v1.13.0
	golang.org/x/crypto v0.0.0-20191122220453-ac88ee75c92c
	golang.org/x/image v0.0.0-20200927104501-e162460cd6b5
	golang.org/x/mod v0.1.1-0.20191209134235-331c550502dd // indirect
	golang.org/x/text v0.3.3 // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	golang.org/x/tools v0.0.0-20200211045251-2de505fc5306 // indirect
	google.golang.org/genproto v0.0.0-20191115221424-83cc0476cb11 // indirect
	google.golang.org/grpc v1.25.1 // indirect
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/go-playground/validator.v9 v9.31.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/yaml.v2 v2.2.6 // indirect
)
