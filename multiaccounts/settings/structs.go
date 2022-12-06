package settings

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

type ValueHandler func(interface{}) (interface{}, error)
type SyncSettingProtobufFactoryInterface func(interface{}, uint64, string) (*common.RawMessage, error)
type SyncSettingProtobufFactoryStruct func(Settings, uint64, string) (*common.RawMessage, error)
type SyncSettingProtobufToValue func(setting *protobuf.SyncSetting) interface{}

// SyncProtobufFactory represents a collection of functionality to generate and parse *protobuf.SyncSetting
type SyncProtobufFactory struct {
	inactive          bool
	fromInterface     SyncSettingProtobufFactoryInterface
	fromStruct        SyncSettingProtobufFactoryStruct
	valueFromProtobuf SyncSettingProtobufToValue
	protobufType      protobuf.SyncSetting_Type
}

func (spf *SyncProtobufFactory) Inactive() bool {
	return spf.inactive
}

func (spf *SyncProtobufFactory) FromInterface() SyncSettingProtobufFactoryInterface {
	return spf.fromInterface
}

func (spf *SyncProtobufFactory) FromStruct() SyncSettingProtobufFactoryStruct {
	return spf.fromStruct
}

func (spf *SyncProtobufFactory) ExtractValueFromProtobuf() SyncSettingProtobufToValue {
	return spf.valueFromProtobuf
}

func (spf *SyncProtobufFactory) SyncSettingProtobufType() protobuf.SyncSetting_Type {
	return spf.protobufType
}

// SyncSettingField represents a binding between a Value and a SettingField
type SyncSettingField struct {
	SettingField
	Value interface{}
}

func (s SyncSettingField) MarshalJSON() ([]byte, error) {
	alias := struct {
		Name  string      `json:"name"`
		Value interface{} `json:"value"`
	}{
		s.reactFieldName,
		s.Value,
	}

	return json.Marshal(alias)
}

// SettingField represents an individual setting in the database, it contains context dependant names and optional
// pre-store value parsing, along with optional *SyncProtobufFactory
type SettingField struct {
	reactFieldName      string
	dBColumnName        string
	fieldName           string
	valueHandler        ValueHandler
	syncProtobufFactory *SyncProtobufFactory
	pairingBootstrap    bool
}

func (s SettingField) GetReactName() string {
	return s.reactFieldName
}

func (s SettingField) GetDBName() string {
	return s.dBColumnName
}

func (s SettingField) ValueHandler() ValueHandler {
	return s.valueHandler
}

func (s SettingField) SyncProtobufFactory() *SyncProtobufFactory {
	return s.syncProtobufFactory
}

// CanSync checks if a SettingField has functions supporting the syncing of
func (s SettingField) CanSync(source SyncSource) bool {
	spf := s.syncProtobufFactory

	if spf == nil {
		return false
	}

	if spf.inactive {
		return false
	}

	switch source {
	case FromInterface:
		return spf.fromInterface != nil
	case FromStruct:
		return spf.fromStruct != nil
	default:
		return false
	}
}

// Settings represents the entire setting row stored in the application db
type Settings struct {
	// required
	// TODO resolve / decide what to do with the db tags
	Address                   types.Address    `json:"address" db:"address"`
	AnonMetricsShouldSend     bool             `json:"anon-metrics/should-send?,omitempty" db:"anonMetricsShouldSend"`
	ChaosMode                 bool             `json:"chaos-mode?,omitempty" db:"chaosMode"`
	Currency                  string           `json:"currency,omitempty" db:"currency"`
	CurrentNetwork            string           `json:"networks/current-network" db:"currentNetwork"`
	CustomBootnodes           *json.RawMessage `json:"custom-bootnodes,omitempty" db:"customBootnodes"`
	CustomBootnodesEnabled    *json.RawMessage `json:"custom-bootnodes-enabled?,omitempty" db:"customBootnodesEnabled"`
	DappsAddress              types.Address    `json:"dapps-address" db:"dappsAddress"`
	DisplayName               string           `json:"display-name" db:"displayName"`
	Bio                       string           `json:"bio,omitempty" db:"bio"`
	EIP1581Address            types.Address    `json:"eip1581-address" db:"EIP1581Address"`
	Fleet                     *string          `json:"fleet,omitempty" db:"fleet"`
	HideHomeTooltip           bool             `json:"hide-home-tooltip?,omitempty" db:"hideHomeTooltip"`
	InstallationID            string           `json:"installation-id" db:"installationID"`
	KeyUID                    string           `json:"key-uid" db:"keyUID"`
	KeycardInstanceUID        string           `json:"keycard-instance-uid,omitempty" db:"keycardInstanceUID"`
	KeycardPairedOn           int64            `json:"keycard-paired-on,omitempty" db:"keycardPairedOn"`
	KeycardPairing            string           `json:"keycard-pairing,omitempty" db:"keycardPairing"`
	LastUpdated               *int64           `json:"last-updated,omitempty" db:"lastUpdated"`
	LatestDerivedPath         uint             `json:"latest-derived-path" db:"latestDerivedPath"`
	LinkPreviewRequestEnabled bool             `json:"link-preview-request-enabled,omitempty" db:"linkPreviewRequestEnabled"`
	LinkPreviewsEnabledSites  *json.RawMessage `json:"link-previews-enabled-sites,omitempty" db:"linkPreviewsEnabledSites"`
	LogLevel                  *string          `json:"log-level,omitempty" db:"logLevel"`
	MessagesFromContactsOnly  bool             `json:"messages-from-contacts-only" db:"messagesFromContactsOnly"`
	Mnemonic                  *string          `json:"mnemonic,omitempty" db:"mnemonic"`
	MutualContactEnabled      bool             `json:"mutual-contact-enabled?" db:"mutualContactEnabled"`
	Name                      string           `json:"name,omitempty" db:"name"`
	Networks                  *json.RawMessage `json:"networks/networks" db:"networks"`
	// NotificationsEnabled indicates whether local notifications should be enabled (android only)
	NotificationsEnabled bool             `json:"notifications-enabled?,omitempty" db:"notificationsEnabled"`
	PhotoPath            string           `json:"photo-path" db:"photoPath"`
	PinnedMailserver     *json.RawMessage `json:"pinned-mailservers,omitempty" db:"pinnedMailserver"`
	PreferredName        *string          `json:"preferred-name,omitempty" db:"preferredName"`
	PreviewPrivacy       bool             `json:"preview-privacy?" db:"previewPrivacy"`
	PublicKey            string           `json:"public-key" db:"publicKey"`
	// PushNotificationsServerEnabled indicates whether we should be running a push notification server
	PushNotificationsServerEnabled bool `json:"push-notifications-server-enabled?,omitempty" db:"pushNotificationsServerEnabled"`
	// PushNotificationsFromContactsOnly indicates whether we should only receive push notifications from contacts
	PushNotificationsFromContactsOnly bool `json:"push-notifications-from-contacts-only?,omitempty" db:"pushNotificationsFromContactsOnly"`
	// PushNotificationsBlockMentions indicates whether we should receive notifications for mentions
	PushNotificationsBlockMentions bool `json:"push-notifications-block-mentions?,omitempty" db:"pushNotificationsBlockMentions"`
	RememberSyncingChoice          bool `json:"remember-syncing-choice?,omitempty" db:"rememberSyncingChoice"`
	// RemotePushNotificationsEnabled indicates whether we should be using remote notifications (ios only for now)
	RemotePushNotificationsEnabled bool             `json:"remote-push-notifications-enabled?,omitempty" db:"remotePushNotificationsEnabled"`
	SigningPhrase                  string           `json:"signing-phrase" db:"signingPhrase"`
	StickerPacksInstalled          *json.RawMessage `json:"stickers/packs-installed,omitempty" db:"stickerPacksInstalled"`
	StickerPacksPending            *json.RawMessage `json:"stickers/packs-pending,omitempty" db:"stickerPacksPending"`
	StickersRecentStickers         *json.RawMessage `json:"stickers/recent-stickers,omitempty" db:"stickersRecentStickers"`
	SyncingOnMobileNetwork         bool             `json:"syncing-on-mobile-network?,omitempty" db:"syncingOnMobileNetwork"`
	// DefaultSyncPeriod is how far back in seconds we should pull messages from a mailserver
	DefaultSyncPeriod uint `json:"default-sync-period" db:"defaultSyncPeriod"`
	// SendPushNotifications indicates whether we should send push notifications for other clients
	SendPushNotifications bool `json:"send-push-notifications?,omitempty" db:"sendPushNotifications"`
	Appearance            uint `json:"appearance" db:"appearance"`
	// ProfilePicturesShowTo indicates to whom the user shows their profile picture to (contacts, everyone)
	ProfilePicturesShowTo ProfilePicturesShowToType `json:"profile-pictures-show-to" db:"profilePicturesShowTo"`
	// ProfilePicturesVisibility indicates who we want to see profile pictures of (contacts, everyone or none)
	ProfilePicturesVisibility      ProfilePicturesVisibilityType `json:"profile-pictures-visibility" db:"profilePicturesVisibility"`
	UseMailservers                 bool                          `json:"use-mailservers?" db:"useMailservers"`
	Usernames                      *json.RawMessage              `json:"usernames,omitempty" db:"usernames"`
	WalletRootAddress              types.Address                 `json:"wallet-root-address,omitempty" db:"walletRootAddress"`
	WalletSetUpPassed              bool                          `json:"wallet-set-up-passed?,omitempty" db:"walletSetUpPassed"`
	WalletVisibleTokens            *json.RawMessage              `json:"wallet/visible-tokens,omitempty" db:"walletVisibleTokens"`
	WakuBloomFilterMode            bool                          `json:"waku-bloom-filter-mode,omitempty" db:"wakuBloomFilterMode"`
	WebViewAllowPermissionRequests bool                          `json:"webview-allow-permission-requests?,omitempty" db:"webViewAllowPermissionRequests"`
	SendStatusUpdates              bool                          `json:"send-status-updates?,omitempty" db:"sendStatusUpdates"`
	CurrentUserStatus              *json.RawMessage              `json:"current-user-status" db:"currentUserStatus"`
	GifRecents                     *json.RawMessage              `json:"gifs/recent-gifs" db:"gifRecents"`
	GifFavorites                   *json.RawMessage              `json:"gifs/favorite-gifs" db:"gifFavorites"`
	OpenseaEnabled                 bool                          `json:"opensea-enabled?,omitempty" db:"openseaEnabled"`
	TelemetryServerURL             string                        `json:"telemetry-server-url,omitempty" db:"telemetryServerURL"`
	LastBackup                     uint64                        `json:"last-backup,omitempty" db:"lastBackup"`
	BackupEnabled                  bool                          `json:"backup-enabled?,omitempty" db:"backupEnabled"`
	AutoMessageEnabled             bool                          `json:"auto-message-enabled?,omitempty" db:"autoMessageEnabled"`
	GifAPIKey                      string                        `json:"gifs/api-key" db:"gifAPIKey"`
	TestNetworksEnabled            bool                          `json:"test-networks-enabled?,omitempty" db:"testNetworksEnabled"`
}

type fieldGroup struct {
	v reflect.Value
	f reflect.StructField
}

// getFieldByName returns a *fieldGroup of any Settings field
func (s Settings) getFieldByName(name string) (*fieldGroup, error) {
	fv := reflect.ValueOf(s).FieldByName(name)
	if !fv.IsValid() {
		return nil, fmt.Errorf("not a Settings field name: %s", name)
	}

	ft, ok := reflect.TypeOf(s).FieldByName(name)
	if !ok {
		return nil, fmt.Errorf("not a Settings field name: %s", name)
	}

	if fv.Kind() == reflect.Ptr {
		fv = fv.Elem()
	}
	return &fieldGroup{v: fv, f: ft}, nil
}

// normaliseFieldValue converts a Settings field reflect.Value data type into a reflect.Kind
// returning the converted reflect.Value
func (s Settings) normaliseFieldValue(fv reflect.Value) (reflect.Value, error) {
	// Check the underlying kind of the Settings field value, convert to the underlying type
	// this datatype normalisation is required because protobufs only accept scala types
	switch fv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fv = fv.Convert(reflect.TypeOf(int64(0)))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		fv = fv.Convert(reflect.TypeOf(uint64(0)))
	case reflect.Array:
		fv = fv.Convert(reflect.ArrayOf(fv.Len(), fv.Type().Elem()))
		zas := reflect.MakeSlice(reflect.SliceOf(fv.Type().Elem()), fv.Len(), fv.Cap())
		reflect.Copy(zas, fv)
		fv = zas
	case reflect.Slice:
		fv = fv.Convert(reflect.SliceOf(fv.Type().Elem()))
	}

	return fv, nil
}

// fieldToPairingBootstrapSetting gets field of Settings with given name to given value.
func (s Settings) fieldToPairingBootstrapSetting(fg *fieldGroup) (*protobuf.PairingBootstrapSetting, error) {
	fv, err := s.normaliseFieldValue(fg.v)
	if err != nil {
		return nil, err
	}

	pbs := new(protobuf.PairingBootstrapSetting)
	pbs.Type = fg.f.Name

	// Skip any nil values
	if !fv.IsValid() {
		return nil, nil
	}

	switch fvi := fv.Interface(); fvi.(type) {
	case bool:
		pbs.Value = &protobuf.PairingBootstrapSetting_Bool{Bool: fvi.(bool)}
	case []byte:
		pbs.Value = &protobuf.PairingBootstrapSetting_Bytes{Bytes: fvi.([]byte)}
	case int64:
		pbs.Value = &protobuf.PairingBootstrapSetting_Int64{Int64: fvi.(int64)}
	case string:
		pbs.Value = &protobuf.PairingBootstrapSetting_String_{String_: fvi.(string)}
	case uint64:
		pbs.Value = &protobuf.PairingBootstrapSetting_Uint64{Uint64: fvi.(uint64)}
	default:
		return nil, fmt.Errorf("unexpected data type %T", fvi)
	}

	return pbs, nil
}

func (s Settings) pairingBootstrapSettingToField(pb *protobuf.PairingBootstrapSetting) (*fieldGroup, error) {
	var v reflect.Value
	switch pbv := pb.GetValue(); pbv.(type) {
	case *protobuf.PairingBootstrapSetting_Bool:
		v = reflect.ValueOf(pb.GetBool())
	case *protobuf.PairingBootstrapSetting_Bytes:
		v = reflect.ValueOf(pb.GetBytes())
	case *protobuf.PairingBootstrapSetting_Int64:
		v = reflect.ValueOf(pb.GetInt64())
	case *protobuf.PairingBootstrapSetting_String_:
		v = reflect.ValueOf(pb.GetString_())
	case *protobuf.PairingBootstrapSetting_Uint64:
		v = reflect.ValueOf(pb.GetUint64())
	default:
		return nil, fmt.Errorf("unexpected data type %T", pbv)
	}

	fg, err := s.getFieldByName(pb.Type)
	if err != nil {
		return nil, err
	}

	if fg.f.Type.Kind() == reflect.Ptr {
		pt := reflect.PointerTo(fg.f.Type.Elem())
		pv := reflect.New(pt.Elem())
		pv.Elem().Set(v)
		v = pv
	}

	if fg.f.Type == v.Type() {
		fg.v = v
		return fg, nil
	}

	switch fg.f.Type.Kind() {
	case reflect.Array:
		a := reflect.New(reflect.ArrayOf(v.Len(), v.Type().Elem())).Elem()
		reflect.Copy(a, v)
		v = a
		fallthrough
	default:
		v = v.Convert(fg.f.Type)
	}

	fg.v = v
	return fg, nil
}

func (s Settings) ToPairingBootstrapSettings() (*protobuf.PairingBootstrapSettings, error) {
	pbs := new(protobuf.PairingBootstrapSettings)

	for _, sf := range SettingFieldRegister {
		if !sf.pairingBootstrap {
			continue
		}

		fg, err := s.getFieldByName(sf.fieldName)
		if err != nil {
			return nil, err
		}

		pb, err := s.fieldToPairingBootstrapSetting(fg)
		if err != nil {
			return nil, err
		}
		if pb == nil {
			continue
		}

		pbs.Settings = append(pbs.Settings, pb)
	}

	return pbs, nil
}

func (s Settings) FromPairingBootstrapSettings(pbs *protobuf.PairingBootstrapSettings) ([]SyncSettingField, error) {
	var out []SyncSettingField

	for _, pb := range pbs.Settings {
		fg, err := s.pairingBootstrapSettingToField(pb)
		if err != nil {
			return nil, err
		}

		fs, err := GetFieldFromFieldName(fg.f.Name)
		if err != nil {
			return nil, err
		}

		ssf := SyncSettingField{
			SettingField: fs,
			Value:        fg.v.Interface(),
		}
		out = append(out, ssf)
	}

	return out, nil
}
