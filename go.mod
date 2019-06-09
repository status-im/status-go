module github.com/status-im/status-go

go 1.12

require (
	github.com/NaySoftware/go-fcm v0.0.0-20190516140123-808e978ddcd2
	github.com/beevik/ntp v0.2.0
	github.com/btcsuite/btcd v0.0.0-20190523000118-16327141da8c
	github.com/btcsuite/btcutil v0.0.0-20190425235716-9e5f4b9a998d
	github.com/ethereum/go-ethereum v1.8.27
	github.com/go-playground/locales v0.12.1 // indirect
	github.com/go-playground/universal-translator v0.16.0 // indirect
	github.com/golang-migrate/migrate v3.5.4+incompatible // indirect
	github.com/golang-migrate/migrate/v4 v4.4.0 // indirect
	github.com/golang/mock v1.2.0
	github.com/golang/protobuf v1.3.1
	github.com/lib/pq v1.0.0
	github.com/libp2p/go-libp2p-core v0.0.3
	github.com/multiformats/go-multiaddr v0.0.4
	github.com/mutecomm/go-sqlcipher v0.0.0-20170920224653-f799951b4ab2
	github.com/pborman/uuid v0.0.0-20170112150404-1b00554d8222
	github.com/russolsen/ohyeah v0.0.0-20160324131710-f4938c005315 // indirect
	github.com/russolsen/same v0.0.0-20160222130632-f089df61f51d // indirect
	github.com/russolsen/transit v0.0.0-20180705123435-0794b4c4505a
	github.com/status-im/doubleratchet v2.0.0+incompatible
	github.com/status-im/migrate/v4 v4.3.1-status
	github.com/status-im/rendezvous v1.3.0
	github.com/status-im/whisper v1.4.13
	github.com/stretchr/testify v1.3.0
	github.com/syndtr/goleveldb v1.0.0
	golang.org/x/crypto v0.0.0-20190530122614-20be4c3c3ed5
	golang.org/x/text v0.3.2
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/go-playground/validator.v9 v9.9.3
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
)

replace github.com/ethereum/go-ethereum v1.8.27 => github.com/status-im/go-ethereum v1.8.27-status.3

replace github.com/NaySoftware/go-fcm => github.com/status-im/go-fcm v1.0.0-status
