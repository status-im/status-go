package pairing

import (
	"encoding/json"
	"errors"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/protocol/protobuf"
)

const keystoreDir = "keystore"

var (
	ErrKeyFileAlreadyExists    = errors.New("key file already exists")
	ErrKeyUIDEmptyAsSender     = errors.New("keyUID must be provided as sender")
	ErrNodeConfigNilAsReceiver = errors.New("node config must be provided as receiver")
	ErrLoggedInKeyUIDConflict  = errors.New("logged in keyUID not same as keyUID in payload")
)

// AccountPayload represents the payload structure a Server handles
type AccountPayload struct {
	keys            map[string][]byte // nolint: structcheck
	multiaccount    *multiaccounts.Account
	password        string
	chatKey         string
	keycardPairings string
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
	lpp := &protobuf.LocalPairingPayload{
		Keys:            ppm.accountKeysToProtobuf(),
		Password:        ppm.password,
		ChatKey:         ppm.chatKey,
		KeycardPairings: ppm.keycardPairings,
	}
	if ppm.multiaccount != nil {
		lpp.Multiaccount = ppm.multiaccount.ToProtobuf()
	}
	return proto.Marshal(lpp)
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
	ppm.chatKey = pb.ChatKey
	ppm.keycardPairings = pb.KeycardPairings
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

type RawMessagesPayload struct {
	rawMessages    []*protobuf.RawMessage
	profileKeypair *accounts.Keypair
	setting        *settings.Settings
}

func NewRawMessagesPayload() *RawMessagesPayload {
	return &RawMessagesPayload{
		setting: new(settings.Settings),
	}
}

// RawMessagePayloadMarshaller is responsible for marshalling and unmarshalling raw message data
type RawMessagePayloadMarshaller struct {
	payload *RawMessagesPayload
}

func NewRawMessagePayloadMarshaller(payload *RawMessagesPayload) *RawMessagePayloadMarshaller {
	return &RawMessagePayloadMarshaller{
		payload: payload,
	}
}

func (rmm *RawMessagePayloadMarshaller) MarshalProtobuf() (data []byte, err error) {
	syncRawMessage := new(protobuf.SyncRawMessage)

	syncRawMessage.RawMessages = rmm.payload.rawMessages
	if rmm.payload.profileKeypair != nil && len(rmm.payload.profileKeypair.KeyUID) > 0 {
		syncRawMessage.SubAccountsJsonBytes, err = json.Marshal(rmm.payload.profileKeypair)
		if err != nil {
			return nil, err
		}
	}
	if !rmm.payload.setting.IsEmpty() {
		syncRawMessage.SettingsJsonBytes, err = json.Marshal(rmm.payload.setting)
		if err != nil {
			return nil, err
		}
	}

	return proto.Marshal(syncRawMessage)
}

func (rmm *RawMessagePayloadMarshaller) UnmarshalProtobuf(data []byte) error {
	syncRawMessage := new(protobuf.SyncRawMessage)
	err := proto.Unmarshal(data, syncRawMessage)
	if err != nil {
		return err
	}
	if syncRawMessage.SubAccountsJsonBytes != nil {
		err = json.Unmarshal(syncRawMessage.SubAccountsJsonBytes, &rmm.payload.profileKeypair)
		if err != nil {
			return err
		}
	}
	if syncRawMessage.SettingsJsonBytes != nil {
		err = json.Unmarshal(syncRawMessage.SettingsJsonBytes, rmm.payload.setting)
		if err != nil {
			return err
		}
	}

	rmm.payload.rawMessages = syncRawMessage.RawMessages
	return nil
}

// InstallationPayloadMounterReceiver represents an InstallationPayload Repository
type InstallationPayloadMounterReceiver struct {
	PayloadMounter
	PayloadReceiver
}

func NewInstallationPayloadMounterReceiver(encryptor *PayloadEncryptor, backend *api.GethStatusBackend, deviceType string) *InstallationPayloadMounterReceiver {
	return &InstallationPayloadMounterReceiver{
		NewInstallationPayloadMounter(encryptor, backend, deviceType),
		NewInstallationPayloadReceiver(encryptor, backend, deviceType),
	}
}

func (i *InstallationPayloadMounterReceiver) LockPayload() {
	i.PayloadMounter.LockPayload()
	i.PayloadReceiver.LockPayload()
}
