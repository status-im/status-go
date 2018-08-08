package params

import "errors"

// Define available fleets.
const (
	FleetUndefined = ""
	FleetBeta      = "eth.beta"
	FleetStaging   = "eth.staging"
)

type cluster struct {
	NetworkID   int      `json:"networkID"`
	StaticNodes []string `json:"staticnodes"`
	BootNodes   []string `json:"bootnodes"`
	MailServers []string `json:"mailservers"` // list of trusted mail servers
}

var ropstenCluster = cluster{
	NetworkID: RopstenNetworkID,
	BootNodes: []string{
		"enode://436cc6f674928fdc9a9f7990f2944002b685d1c37f025c1be425185b5b1f0900feaf1ccc2a6130268f9901be4a7d252f37302c8335a2c1a62736e9232691cc3a@174.138.105.243:30404", // boot-01.do-ams3.eth.beta
		"enode://5395aab7833f1ecb671b59bf0521cf20224fe8162fc3d2675de4ee4d5636a75ec32d13268fc184df8d1ddfa803943906882da62a4df42d4fccf6d17808156a87@206.189.243.57:30404",  // boot-02.do-ams3.eth.beta
		"enode://7427dfe38bd4cf7c58bb96417806fab25782ec3e6046a8053370022cbaa281536e8d64ecd1b02e1f8f72768e295d06258ba43d88304db068e6f2417ae8bcb9a6@104.154.88.123:30404",  // boot-01.gc-us-central1-a.eth.beta
		"enode://ebefab39b69bbbe64d8cd86be765b3be356d8c4b24660f65d493143a0c44f38c85a257300178f7845592a1b0332811542e9a58281c835babdd7535babb64efc1@35.202.99.224:30404",   // boot-02.gc-us-central1-a.eth.beta
	},
	StaticNodes: []string{
		"enode://a6a2a9b3a7cbb0a15da74301537ebba549c990e3325ae78e1272a19a3ace150d03c184b8ac86cc33f1f2f63691e467d49308f02d613277754c4dccd6773b95e8@206.189.243.176:30304", // node-01.do-ams3.eth.beta
		"enode://207e53d9bf66be7441e3daba36f53bfbda0b6099dba9a865afc6260a2d253fb8a56a72a48598a4f7ba271792c2e4a8e1a43aaef7f34857f520c8c820f63b44c8@35.224.15.65:30304",    // node-01.gc-us-central1-a.eth.beta
	},
	MailServers: []string{
		"enode://c42f368a23fa98ee546fd247220759062323249ef657d26d357a777443aec04db1b29a3a22ef3e7c548e18493ddaf51a31b0aed6079bd6ebe5ae838fcfaf3a49@206.189.243.162:30504", // mail-01.do-ams3.eth.beta
		"enode://7aa648d6e855950b2e3d3bf220c496e0cae4adfddef3e1e6062e6b177aec93bc6cdcf1282cb40d1656932ebfdd565729da440368d7c4da7dbd4d004b1ac02bf8@206.189.243.169:30504", // mail-02.do-ams3.eth.beta
		"enode://8a64b3c349a2e0ef4a32ea49609ed6eb3364be1110253c20adc17a3cebbc39a219e5d3e13b151c0eee5d8e0f9a8ba2cd026014e67b41a4ab7d1d5dd67ca27427@206.189.243.168:30504", // mail-03.do-ams3.eth.beta
		"enode://7de99e4cb1b3523bd26ca212369540646607c721ad4f3e5c821ed9148150ce6ce2e72631723002210fac1fd52dfa8bbdf3555e05379af79515e1179da37cc3db@35.188.19.210:30504",   // mail-01.gc-us-central1-a.eth.beta
		"enode://015e22f6cd2b44c8a51bd7a23555e271e0759c7d7f52432719665a74966f2da456d28e154e836bee6092b4d686fe67e331655586c57b718be3997c1629d24167@35.226.21.19:30504",    // mail-02.gc-us-central1-a.eth.beta
		"enode://531e252ec966b7e83f5538c19bf1cde7381cc7949026a6e499b6e998e695751aadf26d4c98d5a4eabfb7cefd31c3c88d600a775f14ed5781520a88ecd25da3c6@35.225.227.79:30504",   // mail-03.gc-us-central1-a.eth.beta
	},
}

var rinkebyCluster = cluster{
	NetworkID: RinkebyNetworkID,
	BootNodes: []string{
		"enode://1b843c7697f6fc42a1f606fb3cfaac54e025f06789dc20ad9278be3388967cf21e3a1b1e4be51faecd66c2c3adef12e942b4fcdeb8727657abe60636efb6224f@206.189.6.46:30404",
		"enode://b29100c8468e3e6604817174a15e4d71627458b0dcdbeea169ab2eb4ab2bbc6f24adbb175826726cec69db8fdba6c0dd60b3da598e530ede562180d300728659@206.189.6.48:30404",
	},
	StaticNodes: []string{
		"enode://ff1d6ac1c1d79fe060137d217ad26e372b6dea3d53690677e231000334f6e71c0b720000b6f79edb1e1100c172c1df85a3f05867e4f0716e7ff7fbc47327898b@51.15.75.244:30303",
		"enode://6a1e9b88da1cb5e55e9174c21d3808800671c342416e90edd181341b5c2192a9a6189a770a69ae7cf24dd97cb1322f9b56d8093549a2bf944b3baaa6ccaa9ba9@51.15.68.93:30303",
		"enode://ba41aa829287a0a9076d9bffed97c8ce2e491b99873288c9e886f16fd575306ac6c656db4fbf814f5a9021aec004ffa9c0ae8650f92fd10c12eeb7c364593eb3@51.15.69.147:30303",
		"enode://28ecf5272b560ca951f4cd7f1eb8bd62da5853b026b46db432c4b01797f5b0114819a090a72acd7f32685365ecd8e00450074fa0673039aefe10f3fb666e0f3f@51.15.76.249:30303",
	},
	MailServers: []string{
		"enode://43829580446ad138386dadb7fa50b6bd4d99f7c28659a0bc08115f8c0380005922a340962496f6af756a42b94a1522baa38a694fa27de59c3a73d4e08d5dbb31@206.189.6.48:30504",
		"enode://70a2004e78399075f566033c42e9a0b1d43c683d4742755bb5457d03191be66a1b48c2b4fb259696839f28646a5828a1958b900860e27897f984ad0fc8482404@206.189.56.154:30504",
	},
}

var mainnetCluster = cluster{
	NetworkID: MainNetworkID,
	BootNodes: []string{
		"enode://436cc6f674928fdc9a9f7990f2944002b685d1c37f025c1be425185b5b1f0900feaf1ccc2a6130268f9901be4a7d252f37302c8335a2c1a62736e9232691cc3a@174.138.105.243:30404", // boot-01.do-ams3.eth.beta
		"enode://5395aab7833f1ecb671b59bf0521cf20224fe8162fc3d2675de4ee4d5636a75ec32d13268fc184df8d1ddfa803943906882da62a4df42d4fccf6d17808156a87@206.189.243.57:30404",  // boot-02.do-ams3.eth.beta
		"enode://7427dfe38bd4cf7c58bb96417806fab25782ec3e6046a8053370022cbaa281536e8d64ecd1b02e1f8f72768e295d06258ba43d88304db068e6f2417ae8bcb9a6@104.154.88.123:30404",  // boot-01.gc-us-central1-a.eth.beta
		"enode://ebefab39b69bbbe64d8cd86be765b3be356d8c4b24660f65d493143a0c44f38c85a257300178f7845592a1b0332811542e9a58281c835babdd7535babb64efc1@35.202.99.224:30404",   // boot-02.gc-us-central1-a.eth.beta
	},
	StaticNodes: []string{
		"enode://a6a2a9b3a7cbb0a15da74301537ebba549c990e3325ae78e1272a19a3ace150d03c184b8ac86cc33f1f2f63691e467d49308f02d613277754c4dccd6773b95e8@206.189.243.176:30304", // node-01.do-ams3.eth.beta
		"enode://207e53d9bf66be7441e3daba36f53bfbda0b6099dba9a865afc6260a2d253fb8a56a72a48598a4f7ba271792c2e4a8e1a43aaef7f34857f520c8c820f63b44c8@35.224.15.65:30304",    // node-01.gc-us-central1-a.eth.beta
	},
	MailServers: []string{
		"enode://c42f368a23fa98ee546fd247220759062323249ef657d26d357a777443aec04db1b29a3a22ef3e7c548e18493ddaf51a31b0aed6079bd6ebe5ae838fcfaf3a49@206.189.243.162:30504", // mail-01.do-ams3.eth.beta
		"enode://7aa648d6e855950b2e3d3bf220c496e0cae4adfddef3e1e6062e6b177aec93bc6cdcf1282cb40d1656932ebfdd565729da440368d7c4da7dbd4d004b1ac02bf8@206.189.243.169:30504", // mail-02.do-ams3.eth.beta
		"enode://8a64b3c349a2e0ef4a32ea49609ed6eb3364be1110253c20adc17a3cebbc39a219e5d3e13b151c0eee5d8e0f9a8ba2cd026014e67b41a4ab7d1d5dd67ca27427@206.189.243.168:30504", // mail-03.do-ams3.eth.beta
		"enode://7de99e4cb1b3523bd26ca212369540646607c721ad4f3e5c821ed9148150ce6ce2e72631723002210fac1fd52dfa8bbdf3555e05379af79515e1179da37cc3db@35.188.19.210:30504",   // mail-01.gc-us-central1-a.eth.beta
		"enode://015e22f6cd2b44c8a51bd7a23555e271e0759c7d7f52432719665a74966f2da456d28e154e836bee6092b4d686fe67e331655586c57b718be3997c1629d24167@35.226.21.19:30504",    // mail-02.gc-us-central1-a.eth.beta
		"enode://531e252ec966b7e83f5538c19bf1cde7381cc7949026a6e499b6e998e695751aadf26d4c98d5a4eabfb7cefd31c3c88d600a775f14ed5781520a88ecd25da3c6@35.225.227.79:30504",   // mail-03.gc-us-central1-a.eth.beta
	},
}

var betaCluster = []cluster{ropstenCluster, rinkebyCluster, mainnetCluster}

var stagingCluster = []cluster{
	{
		NetworkID: MainNetworkID,
		BootNodes: []string{
			"enode://10a78c17929a7019ef4aa2249d7302f76ae8a06f40b2dc88b7b31ebff4a623fbb44b4a627acba296c1ced3775d91fbe18463c15097a6a36fdb2c804ff3fc5b35@35.238.97.234:30404",   // boot-01.gc-us-central1-a.eth.staging
			"enode://f79fb3919f72ca560ad0434dcc387abfe41e0666201ebdada8ede0462454a13deb05cda15f287d2c4bd85da81f0eb25d0a486bbbc8df427b971ac51533bd00fe@174.138.107.239:30404", // boot-01.do-ams3.eth.staging
		},
	},
	{
		NetworkID: RopstenNetworkID,
		BootNodes: []string{
			"enode://10a78c17929a7019ef4aa2249d7302f76ae8a06f40b2dc88b7b31ebff4a623fbb44b4a627acba296c1ced3775d91fbe18463c15097a6a36fdb2c804ff3fc5b35@35.238.97.234:30404",   // boot-01.gc-us-central1-a.eth.staging
			"enode://f79fb3919f72ca560ad0434dcc387abfe41e0666201ebdada8ede0462454a13deb05cda15f287d2c4bd85da81f0eb25d0a486bbbc8df427b971ac51533bd00fe@174.138.107.239:30404", // boot-01.do-ams3.eth.staging
		},
	},
}

// clusterForFleet returns a cluster for a given fleet.
func clusterForFleet(fleet string) ([]cluster, error) {
	switch fleet {
	case FleetStaging:
		return stagingCluster, nil
	case FleetBeta:
		return betaCluster, nil
	default:
		return nil, errors.New("fleet could not be found")
	}
}
