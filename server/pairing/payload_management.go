package pairing

import (
	"errors"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/protocol/protobuf"
)

const keystoreDir = "keystore"

var (
	// TODO add validation on config to ensure required fields have valid values

	ErrKeyFileAlreadyExists    = errors.New("key file already exists")
	ErrKeyUIDEmptyAsSender     = errors.New("keyUID must be provided as sender")
	ErrNodeConfigNilAsReceiver = errors.New("node config must be provided as receiver")
	ErrLoggedInKeyUIDConflict  = errors.New("logged in keyUID not same as keyUID in payload")
)

// AccountPayload represents the payload structure a Server handles
type AccountPayload struct {
	keys         map[string][]byte
	multiaccount *multiaccounts.Account
	password     string
	//flag if account already exist before sync account
	exist bool
}

// AccountPayloadMarshaller is responsible for marshalling and unmarshalling Server payload data
type AccountPayloadMarshaller struct {
	logger *zap.Logger
	*AccountPayload
}

func NewPairingPayloadMarshaller(ap *AccountPayload, logger *zap.Logger) *AccountPayloadMarshaller {
	return &AccountPayloadMarshaller{logger: logger, AccountPayload: ap}
}

func (ppm *AccountPayloadMarshaller) MarshalProtobuf() ([]byte, error) {
	return proto.Marshal(&protobuf.LocalPairingPayload{
		Keys:         ppm.accountKeysToProtobuf(),
		Multiaccount: ppm.multiaccount.ToProtobuf(),
		Password:     ppm.password,
	})
}

func (ppm *AccountPayloadMarshaller) accountKeysToProtobuf() []*protobuf.LocalPairingPayload_Key {
	var keys []*protobuf.LocalPairingPayload_Key
	for name, data := range ppm.keys {
		keys = append(keys, &protobuf.LocalPairingPayload_Key{Name: name, Data: data})
	}
	return keys
}

func (ppm *AccountPayloadMarshaller) UnmarshalProtobuf(data []byte) error {
	l := ppm.logger.Named("UnmarshalProtobuf()")
	l.Debug("fired")

	pb := new(protobuf.LocalPairingPayload)
	err := proto.Unmarshal(data, pb)
	l.Debug(
		"after protobuf.LocalPairingPayload",
		zap.Any("pb", pb),
		zap.Any("pb.Multiaccount", pb.Multiaccount),
		zap.Any("pb.Keys", pb.Keys),
	)
	if err != nil {
		return err
	}

	ppm.accountKeysFromProtobuf(pb.Keys)
	ppm.multiaccountFromProtobuf(pb.Multiaccount)
	ppm.password = pb.Password
	return nil
}

func (ppm *AccountPayloadMarshaller) accountKeysFromProtobuf(pbKeys []*protobuf.LocalPairingPayload_Key) {
	l := ppm.logger.Named("accountKeysFromProtobuf()")
	l.Debug("fired")

	if ppm.keys == nil {
		ppm.keys = make(map[string][]byte)
	}

	for _, key := range pbKeys {
		ppm.keys[key.Name] = key.Data
	}
	l.Debug(
		"after for _, key := range pbKeys",
		zap.Any("pbKeys", pbKeys),
		zap.Any("accountPayloadMarshaller.keys", ppm.keys),
	)
}

func (ppm *AccountPayloadMarshaller) multiaccountFromProtobuf(pbMultiAccount *protobuf.MultiAccount) {
	ppm.multiaccount = new(multiaccounts.Account)
	ppm.multiaccount.FromProtobuf(pbMultiAccount)
}

// InstallationPayloadMounterReceiver represents an InstallationPayload Repository
type InstallationPayloadMounterReceiver struct {
	*InstallationPayloadMounter
	*InstallationPayloadReceiver
}

func NewInstallationPayloadMounterReceiver(logger *zap.Logger, encryptor *PayloadEncryptor, backend *api.GethStatusBackend, deviceType string) *InstallationPayloadMounterReceiver {
	l := logger.Named("InstallationPayloadMounterReceiver")
	return &InstallationPayloadMounterReceiver{
		NewInstallationPayloadMounter(l, encryptor, backend, deviceType),
		NewInstallationPayloadReceiver(l, encryptor, backend, deviceType),
	}
}

func (i *InstallationPayloadMounterReceiver) LockPayload() {
	i.InstallationPayloadMounter.LockPayload()
	i.InstallationPayloadReceiver.LockPayload()
}
