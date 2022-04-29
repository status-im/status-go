module github.com/status-im/status-go

go 1.13

replace github.com/ethereum/go-ethereum v1.10.16 => github.com/status-im/go-ethereum v1.10.4-status.4

replace github.com/Sirupsen/logrus v1.4.2 => github.com/sirupsen/logrus v1.4.2

replace github.com/docker/docker => github.com/docker/engine v1.4.2-0.20190717161051-705d9623b7c1

replace github.com/nfnt/resize => github.com/status-im/resize v0.0.0-20201215164250-7c6d9f0d3088

replace github.com/forPelevin/gomoji => github.com/status-im/gomoji v1.1.3-0.20220213022530-e5ac4a8732d4

replace github.com/raulk/go-watchdog v1.2.0 => github.com/status-im/go-watchdog v1.2.0-ios-nolibproc

require (
	github.com/anacrolix/torrent v1.41.0
	github.com/beevik/ntp v0.2.0
	github.com/btcsuite/btcutil v1.0.3-0.20201208143702-a53e38424cce
	github.com/cenkalti/backoff/v3 v3.2.2
	github.com/davecgh/go-spew v1.1.1
	github.com/deckarep/golang-set v1.8.0
	github.com/ethereum/go-ethereum v1.10.16
	github.com/forPelevin/gomoji v1.1.2
	github.com/go-playground/universal-translator v0.17.0 // indirect
	github.com/golang-migrate/migrate/v4 v4.8.0 // indirect
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.2
	github.com/google/uuid v1.3.0
	github.com/imdario/mergo v0.3.12
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-ds-sql v0.3.0
	github.com/ipfs/go-log v1.0.5
	github.com/jinzhu/copier v0.0.0-20190924061706-b57f9002281a
	github.com/keighl/metabolize v0.0.0-20150915210303-97ab655d4034
	github.com/kilic/bls12-381 v0.0.0-20200607163746-32e1441c8a9f
	github.com/leodido/go-urn v1.2.0 // indirect
	github.com/lib/pq v1.9.0
	github.com/libp2p/go-libp2p v0.18.0
	github.com/libp2p/go-libp2p-core v0.14.0
	github.com/libp2p/go-libp2p-peerstore v0.6.0
	github.com/libp2p/go-libp2p-pubsub v0.6.1
	github.com/lucasb-eyer/go-colorful v1.0.3
	github.com/mat/besticon v0.0.0-20210314201728-1579f269edb7
	github.com/multiformats/go-multiaddr v0.5.0
	github.com/multiformats/go-multibase v0.0.3
	github.com/multiformats/go-multihash v0.0.15
	github.com/multiformats/go-varint v0.0.6
	github.com/mutecomm/go-sqlcipher v0.0.0-20190227152316-55dbde17881f
	github.com/nfnt/resize v0.0.0-00010101000000-000000000000
	github.com/okzk/sdnotify v0.0.0-20180710141335-d9becc38acbd
	github.com/oliamb/cutter v0.2.2
	github.com/pborman/uuid v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.1
	github.com/russolsen/ohyeah v0.0.0-20160324131710-f4938c005315 // indirect
	github.com/russolsen/same v0.0.0-20160222130632-f089df61f51d // indirect
	github.com/russolsen/transit v0.0.0-20180705123435-0794b4c4505a
	github.com/status-im/doubleratchet v3.0.0+incompatible
	github.com/status-im/go-waku v0.0.0-20220403002242-f1a40fad73c3
	github.com/status-im/go-waku-rendezvous v0.0.0-20211018070416-a93f3b70c432
	github.com/status-im/markdown v0.0.0-20210405121740-32e5a5055fb6
	github.com/status-im/migrate/v4 v4.6.2-status.2
	github.com/status-im/rendezvous v1.3.5-0.20220406135049-e84f589e197a
	github.com/status-im/status-go/extkeys v1.1.2
	github.com/status-im/tcp-shaker v0.0.0-20191114194237-215893130501
	github.com/status-im/zxcvbn-go v0.0.0-20220311183720-5e8676676857
	github.com/stretchr/testify v1.7.1
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
	github.com/tsenart/tb v0.0.0-20181025101425-0d2499c8b6e9
	github.com/vacp2p/mvds v0.0.24-0.20201124060106-26d8e94130d8
	github.com/wealdtech/go-ens/v3 v3.5.0
	github.com/wealdtech/go-multicodec v1.4.0
	github.com/xeipuuv/gojsonschema v1.2.0
	github.com/zenthangplus/goccm v0.0.0-20211005163543-2f2e522aca15
	go.uber.org/zap v1.21.0
	golang.org/x/crypto v0.0.0-20220131195533-30dcbda58838
	golang.org/x/image v0.0.0-20210220032944-ac19c3e999fb
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/go-playground/validator.v9 v9.31.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	olympos.io/encoding/edn v0.0.0-20201019073823-d3554ca0b0a3
)
