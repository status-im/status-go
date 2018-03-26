package params

type subClusterData struct {
	Number      int      `json:"number"`
	Hash        string   `json:"hash"`
	StaticNodes []string `json:"staticnodes"`
}

type clusterData struct {
	NetworkID int            `json:"networkID"`
	Prod      subClusterData `json:"prod"`
	Dev       subClusterData `json:"dev"`
}

var ropstenCluster = clusterData{
	NetworkID: 3,
	Prod: subClusterData{StaticNodes: []string{
		"enode://dffef3874011709b12d1e540d83ddb19a9db8614ad9151d05bcf813585e45cbebba5aaea223fe315786c401d8cecb1ad2de9f179680c536ea30311fb21fa934b@188.166.100.178:30303",
		"enode://03f3661686d30509d621dbe5ee2e3082923f25e94fd41a2dd8dd34bb12a0c4e8fbde52247c6c55e86dc209a8e7c4a5ae56058c65f7b01734d3ab73818b44e2a3@188.166.33.47:30303",
	}},
	Dev: subClusterData{StaticNodes: []string{
		"enode://dffef3874011709b12d1e540d83ddb19a9db8614ad9151d05bcf813585e45cbebba5aaea223fe315786c401d8cecb1ad2de9f179680c536ea30311fb21fa934b@188.166.100.178:30303",
		"enode://03f3661686d30509d621dbe5ee2e3082923f25e94fd41a2dd8dd34bb12a0c4e8fbde52247c6c55e86dc209a8e7c4a5ae56058c65f7b01734d3ab73818b44e2a3@188.166.33.47:30303",
	}},
}

var rinkebyCluster = clusterData{
	NetworkID: 4,
	Prod: subClusterData{StaticNodes: []string{
		"enode://fda3f6273a0f2da4ac5858d1f52e5afaf9def281121be3d37558c67d4d9ca26c6ad7a0520b2cd7454120fb770e86d5760487c9924b2166e65485f606e56d60fc@51.15.69.144:30303",
		"enode://ba41aa829287a0a9076d9bffed97c8ce2e491b99873288c9e886f16fd575306ac6c656db4fbf814f5a9021aec004ffa9c0ae8650f92fd10c12eeb7c364593eb3@51.15.69.147:30303",
		"enode://28ecf5272b560ca951f4cd7f1eb8bd62da5853b026b46db432c4b01797f5b0114819a090a72acd7f32685365ecd8e00450074fa0673039aefe10f3fb666e0f3f@51.15.76.249:30303",
	}},
	Dev: subClusterData{StaticNodes: []string{
		"enode://7512c8f6e7ffdcc723cf77e602a1de9d8cc2e8ad35db309464819122cd773857131aee390fec33894db13da730c8432bb248eed64039e3810e156e979b2847cb@51.15.78.243:30303",
		"enode://1cc27a5a41130a5c8b90db5b2273dc28f7b56f3edfc0dcc57b665d451274b26541e8de49ea7a074281906a82209b9600239c981163b6ff85c3038a8e2bc5d8b8@51.15.68.93:30303",
		"enode://798d17064141b8f88df718028a8272b943d1cb8e696b3dab56519c70b77b1d3469b56b6f4ce3788457646808f5c7299e9116626f2281f30b959527b969a71e4f@51.15.75.244:30303",
	}},
}

var mainnetCluster = clusterData{
	NetworkID: 1,
	Prod:      subClusterData{},
	Dev: subClusterData{StaticNodes: []string{
		"enode://3aeaff0868b19e03fabe33e6e0fcc821094e1601be44edd6f45e3f0171ed964e13623e49987bddd6c517304d2a45dfe66da51e47b2e11d59c4b30cd6094db43d@163.172.176.22:30303",
		"enode://687343483ca41132a16c9ab67b49e9997a34ec38ddb6dd60bf45f9a0ea4c50362f902553d813af44ab1cdb246fc384d4c74b4437c15cefe3bb0e87b399dbb5bb@163.172.176.22:30403",
		"enode://2a3d6c1c86546831e5bb2684ff0ed6d931bdacf3c6cd344706452a1e78c41442d38c62317096175dcea6517959f40ac789f76356348e0a17ee53563cbdf2db48@163.172.176.22:30503",
		"enode://71bb01b58165e3262aea2d3b06dbf9abb8d5512d96e5000e7e41ab2138b47be685935d3eb119fc25e1413db00d8db231fd9d59555a1cd75229821559b6a4eb51@51.15.85.243:30303",
		"enode://7afd119c549a7ab02b3f7bd77ef3490b6d660d5c49d0734a0c8bb23195ced4ace0bf5cde673cd5cfd07dd8d759277f3d8408eb73dc3c217bbe00f0027d06eee9@51.15.85.243:30403",
		"enode://da8af0869e4e8047f21c1ac016b94a7b7d8e935dddd28d4272f88a1ceaee7c15e7deec9b6fd195ed3bc43748893111ebf2b2479ff44a8025ab8d598f3c97b589@51.15.85.243:30503",
		"enode://7ebaa6a8ce2547f10e34fab9cc5626b86d67934a86e1fb36145c0b89fcc7b9315dd6d0a8cc5808d11a55bdc14c78ff675ca956dfec53837b4f1a97392b15ec23@51.15.35.110:30303",
	}},
}

var defaultClusters = []clusterData{ropstenCluster, rinkebyCluster, mainnetCluster}
