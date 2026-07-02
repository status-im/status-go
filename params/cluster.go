package params

import (
	"crypto/ecdsa"
	"encoding/json"
	"os"

	pkgerrors "github.com/pkg/errors"

	"github.com/status-im/status-go/internal/crypto"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/pkg/messaging/waku/fleets"
)

// Fleet names. The fleet definitions now live in pkg/messaging/waku/fleets,
// behind the Waku layer. These constants and the helpers below are thin
// re-exports kept for back-compat while callers migrate off params.* (pm#380).
const (
	FleetUndefined     = fleets.Undefined
	FleetStatusStaging = fleets.StatusStaging
	FleetStatusProd    = fleets.StatusProd
	FleetWakuSandbox   = fleets.WakuSandbox
	FleetWakuTest      = fleets.WakuTest
)

// FleetInfo and FleetsMap alias the types now owned by the fleets package.
type FleetInfo = fleets.Info
type FleetsMap = fleets.Map

var defaultPushNotificationServers = []*ecdsa.PublicKey{
	crypto.MustDecodePubkeyString("04401ba5eda402678dc78a0a40fd0795f4ea8b1e34972c4d15cf33ac01292341c89f0cbc637fa9f7a3ffe0b9dfe90e9cdae7a14925500ab01b6a91c67bae42a97a"),
	crypto.MustDecodePubkeyString("04181141b1d111908aaf05f4788e6778ec07073a1d4e1ce43c73815c40ee4e7345a1cbf5a90a45f601bf3763f12be63b01624ba1f36eeb9572455e7034b8f9f2c4"),
	crypto.MustDecodePubkeyString("045ffc34d5ffda180d94cd3974d9ed2bb082ede68f342babdbe801ceffb7da902087d43f9aa961c7b85029358874c08ef04ecad9f1d95a1f0e448cbdd5d04350c7"),
}

// LoadWakuFleetsFromFile replaces the supported fleets with definitions read
// from a JSON file (whose structure must match FleetsMap).
func LoadWakuFleetsFromFile(filepath string) error {
	return fleets.LoadFromFile(filepath)
}

func loadPushFleetsFromFile(filepath string) ([]string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		err = pkgerrors.Wrap(err, "failed to open push notifications json file")
		return nil, err
	}

	defer file.Close()

	var overridePushNotificationsServers []string
	decoder := json.NewDecoder(file)

	err = decoder.Decode(&overridePushNotificationsServers)
	if err != nil {
		err = pkgerrors.Wrap(err, "failed to decode push notifications json file")
		return nil, err
	}

	return overridePushNotificationsServers, err
}

func LoadPushFleetsFromFile(filepath string) error {
	pushNotifications, err := loadPushFleetsFromFile(filepath)
	if err != nil {
		return err
	}

	defaultPushNotificationServers = make([]*ecdsa.PublicKey, len(pushNotifications))
	for i, pushNotification := range pushNotifications {
		publicKey, err := crypto.DecodePubkeyString(pushNotification)
		if err != nil {
			return err
		}
		defaultPushNotificationServers[i] = publicKey
	}

	return nil
}

func DefaultWakuNodes(fleet string) []string {
	return fleets.WakuNodes(fleet)
}

func DefaultDiscV5Nodes(fleet string) []string {
	return fleets.DiscV5Nodes(fleet)
}

func DefaultClusterID(fleet string) uint16 {
	return fleets.ClusterID(fleet)
}

func IsFleetSupported(fleet string) bool {
	return fleets.IsSupported(fleet)
}

func GetSupportedFleets() FleetsMap {
	return fleets.Supported()
}

func DefaultStoreNodes(fleet string) []messagingtypes.StoreNode {
	return fleets.StoreNodes(fleet)
}

func DefaultPushNotificationServers() []*ecdsa.PublicKey {
	servers := make([]*ecdsa.PublicKey, len(defaultPushNotificationServers))
	copy(servers, defaultPushNotificationServers)
	return servers
}

func DefaultClusterConfig(fleet string) ClusterConfig {
	return ClusterConfig{
		Enabled:              true,
		Fleet:                fleet,
		WakuNodes:            DefaultWakuNodes(fleet),
		DiscV5BootstrapNodes: DefaultDiscV5Nodes(fleet),
		ClusterID:            DefaultClusterID(fleet),
	}
}
