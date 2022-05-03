module github.com/status-im/status-go

go 1.18

replace github.com/ethereum/go-ethereum v1.10.16 => github.com/status-im/go-ethereum v1.10.4-status.4

replace github.com/Sirupsen/logrus v1.4.2 => github.com/sirupsen/logrus v1.4.2

replace github.com/docker/docker => github.com/docker/engine v1.4.2-0.20190717161051-705d9623b7c1

replace github.com/nfnt/resize => github.com/status-im/resize v0.0.0-20201215164250-7c6d9f0d3088

replace github.com/forPelevin/gomoji => github.com/status-im/gomoji v1.1.3-0.20220213022530-e5ac4a8732d4

replace github.com/raulk/go-watchdog v1.2.0 => github.com/status-im/go-watchdog v1.2.0-ios-nolibproc

require (
	github.com/anacrolix/torrent v1.41.0
	github.com/beevik/ntp v0.2.0
	github.com/cenkalti/backoff/v3 v3.2.2
	github.com/davecgh/go-spew v1.1.1
	github.com/deckarep/golang-set v1.8.0
	github.com/ethereum/go-ethereum v1.10.16
	github.com/forPelevin/gomoji v1.1.2
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
	google.golang.org/protobuf v1.27.1
	gopkg.in/go-playground/validator.v9 v9.31.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	olympos.io/encoding/edn v0.0.0-20201019073823-d3554ca0b0a3
)

require (
	github.com/PuerkitoBio/goquery v1.6.1 // indirect
	github.com/RoaringBitmap/roaring v0.9.4 // indirect
	github.com/StackExchange/wmi v0.0.0-20210224194228-fe8f1750fd46 // indirect
	github.com/VictoriaMetrics/fastcache v1.6.0 // indirect
	github.com/anacrolix/chansync v0.3.0 // indirect
	github.com/anacrolix/confluence v1.9.0 // indirect
	github.com/anacrolix/dht/v2 v2.15.2-0.20220123034220-0538803801cb // indirect
	github.com/anacrolix/envpprof v1.1.1 // indirect
	github.com/anacrolix/go-libutp v1.2.0 // indirect
	github.com/anacrolix/log v0.10.1-0.20220123034749-3920702c17f8 // indirect
	github.com/anacrolix/missinggo v1.3.0 // indirect
	github.com/anacrolix/missinggo/perf v1.0.0 // indirect
	github.com/anacrolix/missinggo/v2 v2.5.2 // indirect
	github.com/anacrolix/mmsg v1.0.0 // indirect
	github.com/anacrolix/multiless v0.2.0 // indirect
	github.com/anacrolix/stm v0.3.0 // indirect
	github.com/anacrolix/sync v0.4.0 // indirect
	github.com/anacrolix/upnp v0.1.3-0.20220123035249-922794e51c96 // indirect
	github.com/anacrolix/utp v0.1.0 // indirect
	github.com/andybalholm/cascadia v1.2.0 // indirect
	github.com/benbjohnson/clock v1.1.0 // indirect
	github.com/benbjohnson/immutable v0.3.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bits-and-blooms/bitset v1.2.0 // indirect
	github.com/bradfitz/iter v0.0.0-20191230175014-e8f45d346db8 // indirect
	github.com/btcsuite/btcd v0.22.0-beta // indirect
	github.com/btcsuite/btcutil v1.0.3-0.20201208143702-a53e38424cce // indirect
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/cheekybits/genny v1.0.0 // indirect
	github.com/containerd/cgroups v0.0.0-20201119153540-4cbc285b3327 // indirect
	github.com/coreos/go-systemd/v22 v22.1.0 // indirect
	github.com/cruxic/go-hmac-drbg v0.0.0-20170206035330-84c46983886d // indirect
	github.com/davidlazar/go-crypto v0.0.0-20200604182044-b73af7476f6c // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/edsrzf/mmap-go v1.0.0 // indirect
	github.com/elastic/gosigar v0.14.1 // indirect
	github.com/fjl/memsize v0.0.0-20190710130421-bcb5799ab5e5 // indirect
	github.com/flynn/noise v1.0.0 // indirect
	github.com/francoispqt/gojay v1.2.13 // indirect
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/gballet/go-libpcsclite v0.0.0-20191108122812-4678299bea08 // indirect
	github.com/go-ole/go-ole v1.2.5 // indirect
	github.com/go-playground/locales v0.13.0 // indirect
	github.com/go-playground/universal-translator v0.17.0 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/go-task/slim-sprig v0.0.0-20210107165309-348f09dbbbc0 // indirect
	github.com/godbus/dbus/v5 v5.0.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-migrate/migrate/v4 v4.8.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/google/gopacket v1.1.19 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-multierror v1.0.0 // indirect
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d // indirect
	github.com/holiman/bloomfilter/v2 v2.0.3 // indirect
	github.com/holiman/uint256 v1.2.0 // indirect
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/huin/goupnp v1.0.2 // indirect
	github.com/ipfs/go-datastore v0.5.1 // indirect
	github.com/ipfs/go-ipfs-util v0.0.2 // indirect
	github.com/ipfs/go-log/v2 v2.5.0 // indirect
	github.com/jackpal/go-nat-pmp v1.0.2 // indirect
	github.com/jbenet/go-temp-err-catcher v0.1.0 // indirect
	github.com/jbenet/goprocess v0.1.4 // indirect
	github.com/karalabe/usb v0.0.0-20211005121534-4c5740d64559 // indirect
	github.com/klauspost/compress v1.13.6 // indirect
	github.com/klauspost/cpuid/v2 v2.0.9 // indirect
	github.com/koron/go-ssdp v0.0.2 // indirect
	github.com/leodido/go-urn v1.2.0 // indirect
	github.com/libp2p/go-buffer-pool v0.0.2 // indirect
	github.com/libp2p/go-cidranger v1.1.0 // indirect
	github.com/libp2p/go-conn-security-multistream v0.3.0 // indirect
	github.com/libp2p/go-eventbus v0.2.1 // indirect
	github.com/libp2p/go-flow-metrics v0.0.3 // indirect
	github.com/libp2p/go-libp2p-asn-util v0.1.0 // indirect
	github.com/libp2p/go-libp2p-blankhost v0.3.0 // indirect
	github.com/libp2p/go-libp2p-connmgr v0.3.1 // indirect
	github.com/libp2p/go-libp2p-discovery v0.6.0 // indirect
	github.com/libp2p/go-libp2p-mplex v0.6.0 // indirect
	github.com/libp2p/go-libp2p-nat v0.1.0 // indirect
	github.com/libp2p/go-libp2p-noise v0.3.0 // indirect
	github.com/libp2p/go-libp2p-pnet v0.2.0 // indirect
	github.com/libp2p/go-libp2p-quic-transport v0.16.1 // indirect
	github.com/libp2p/go-libp2p-resource-manager v0.1.5 // indirect
	github.com/libp2p/go-libp2p-swarm v0.10.2 // indirect
	github.com/libp2p/go-libp2p-tls v0.3.1 // indirect
	github.com/libp2p/go-libp2p-transport-upgrader v0.7.1 // indirect
	github.com/libp2p/go-libp2p-yamux v0.8.2 // indirect
	github.com/libp2p/go-mplex v0.6.0 // indirect
	github.com/libp2p/go-msgio v0.1.0 // indirect
	github.com/libp2p/go-nat v0.1.0 // indirect
	github.com/libp2p/go-netroute v0.2.0 // indirect
	github.com/libp2p/go-openssl v0.0.7 // indirect
	github.com/libp2p/go-reuseport v0.1.0 // indirect
	github.com/libp2p/go-reuseport-transport v0.1.0 // indirect
	github.com/libp2p/go-stream-muxer-multistream v0.4.0 // indirect
	github.com/libp2p/go-tcp-transport v0.5.1 // indirect
	github.com/libp2p/go-ws-transport v0.6.1-0.20220221074654-eeaddb3c061d // indirect
	github.com/libp2p/go-yamux/v3 v3.0.2 // indirect
	github.com/lucas-clemente/quic-go v0.25.0 // indirect
	github.com/marten-seemann/qtls-go1-16 v0.1.4 // indirect
	github.com/marten-seemann/qtls-go1-17 v0.1.0 // indirect
	github.com/marten-seemann/qtls-go1-18 v0.1.0-beta.1 // indirect
	github.com/marten-seemann/tcp v0.0.0-20210406111302-dfbc87cc63fd // indirect
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/miekg/dns v1.1.43 // indirect
	github.com/mikioh/tcpinfo v0.0.0-20190314235526-30a79bb1804b // indirect
	github.com/mikioh/tcpopt v0.0.0-20190314235656-172688c1accc // indirect
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1 // indirect
	github.com/minio/sha256-simd v1.0.0 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/mschoch/smat v0.2.0 // indirect
	github.com/multiformats/go-base32 v0.0.3 // indirect
	github.com/multiformats/go-base36 v0.1.0 // indirect
	github.com/multiformats/go-multiaddr-dns v0.3.1 // indirect
	github.com/multiformats/go-multiaddr-fmt v0.1.0 // indirect
	github.com/multiformats/go-multistream v0.2.2 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/opencontainers/runtime-spec v1.0.2 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/pbnjay/memory v0.0.0-20210728143218-7b4eea64cf58 // indirect
	github.com/peterh/liner v1.2.1 // indirect
	github.com/pion/datachannel v1.5.2 // indirect
	github.com/pion/dtls/v2 v2.1.2 // indirect
	github.com/pion/ice/v2 v2.1.20 // indirect
	github.com/pion/interceptor v0.1.7 // indirect
	github.com/pion/logging v0.2.2 // indirect
	github.com/pion/mdns v0.0.5 // indirect
	github.com/pion/randutil v0.1.0 // indirect
	github.com/pion/rtcp v1.2.9 // indirect
	github.com/pion/rtp v1.7.4 // indirect
	github.com/pion/sctp v1.8.2 // indirect
	github.com/pion/sdp/v3 v3.0.4 // indirect
	github.com/pion/srtp/v2 v2.0.5 // indirect
	github.com/pion/stun v0.3.5 // indirect
	github.com/pion/transport v0.13.0 // indirect
	github.com/pion/turn/v2 v2.0.6 // indirect
	github.com/pion/udp v0.1.1 // indirect
	github.com/pion/webrtc/v3 v3.1.24-0.20220208053747-94262c1b2b38 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.30.0 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/prometheus/tsdb v0.10.0 // indirect
	github.com/raulk/clock v1.1.0 // indirect
	github.com/raulk/go-watchdog v1.2.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20200410134404-eec4a21b6bb0 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/rjeczalik/notify v0.9.2 // indirect
	github.com/rs/cors v1.7.0 // indirect
	github.com/rs/dnscache v0.0.0-20210201191234-295bba877686 // indirect
	github.com/russolsen/ohyeah v0.0.0-20160324131710-f4938c005315 // indirect
	github.com/russolsen/same v0.0.0-20160222130632-f089df61f51d // indirect
	github.com/shirou/gopsutil v3.21.5+incompatible // indirect
	github.com/shopspring/decimal v0.0.0-20180709203117-cd690d0c9e24 // indirect
	github.com/spacemonkeygo/spacelog v0.0.0-20180420211403-2296661a0572 // indirect
	github.com/status-im/go-discover v0.0.0-20220220162124-91b97a3e0efe // indirect
	github.com/status-im/go-multiaddr-ethv4 v1.2.1 // indirect
	github.com/status-im/keycard-go v0.0.0-20200402102358-957c09536969 // indirect
	github.com/tklauser/go-sysconf v0.3.6 // indirect
	github.com/tklauser/numcpus v0.2.2 // indirect
	github.com/tyler-smith/go-bip39 v1.1.0 // indirect
	github.com/whyrusleeping/multiaddr-filter v0.0.0-20160516205228-e903e4adabd7 // indirect
	github.com/whyrusleeping/timecache v0.0.0-20160911033111-cfcb2f1abfee // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	go.etcd.io/bbolt v1.3.6 // indirect
	go.opencensus.io v0.23.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	golang.org/x/mod v0.4.2 // indirect
	golang.org/x/net v0.0.0-20220127200216-cd36cc0744dd // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/sys v0.0.0-20211216021012-1d35b9e2eb4e // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	golang.org/x/tools v0.1.5 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/urfave/cli.v1 v1.20.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	modernc.org/libc v1.11.82 // indirect
	modernc.org/mathutil v1.4.1 // indirect
	modernc.org/memory v1.0.5 // indirect
	modernc.org/sqlite v1.14.2-0.20211125151325-d4ed92c0a70f // indirect
	zombiezen.com/go/sqlite v0.8.0 // indirect
)
