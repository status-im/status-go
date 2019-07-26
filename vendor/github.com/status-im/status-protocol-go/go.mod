module github.com/status-im/status-protocol-go

go 1.12

require (
	github.com/aristanetworks/goarista v0.0.0-20190704150520-f44d68189fd7 // indirect
	github.com/deckarep/golang-set v1.7.1 // indirect
	github.com/ethereum/go-ethereum v1.8.27
	github.com/golang/protobuf v1.3.2
	github.com/jinzhu/copier v0.0.0-20190625015134-976e0346caa8
	github.com/leodido/go-urn v1.1.0 // indirect
	github.com/mutecomm/go-sqlcipher v0.0.0-20170920224653-f799951b4ab2
	github.com/pkg/errors v0.8.1
	github.com/rs/cors v1.6.0 // indirect
	github.com/russolsen/transit v0.0.0-20180705123435-0794b4c4505a
	github.com/status-im/doubleratchet v2.0.0+incompatible
	github.com/status-im/migrate/v4 v4.3.1-status
	github.com/status-im/status-go v0.29.0-beta.3
	github.com/status-im/whisper v1.4.14
	github.com/stretchr/testify v1.3.0
	github.com/vacp2p/mvds v0.0.19
	go.uber.org/atomic v1.4.0 // indirect
	go.uber.org/multierr v1.1.0 // indirect
	go.uber.org/zap v1.10.0
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4
	golang.org/x/net v0.0.0-20190628185345-da137c7871d7 // indirect
	gopkg.in/go-playground/validator.v9 v9.29.0 // indirect
)

replace github.com/ethereum/go-ethereum v1.8.27 => github.com/status-im/go-ethereum v1.8.27-status.4
