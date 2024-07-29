package mailservers

import "github.com/status-im/status-go/params"

func DefaultMailserversByFleet(fleet string) []Mailserver {
	var items []Mailserver
	for _, ms := range DefaultMailservers() {
		if ms.Fleet == fleet {
			items = append(items, ms)
		}
	}
	return items
}

func DefaultMailservers() []Mailserver {

	return []Mailserver{
		Mailserver{
			ID:      "mail-01.ac-cn-hongkong-c.eth.prod",
			Address: "enode://606ae04a71e5db868a722c77a21c8244ae38f1bd6e81687cc6cfe88a3063fa1c245692232f64f45bd5408fed5133eab8ed78049332b04f9c110eac7f71c1b429@47.75.247.214:443",
			Fleet:   params.FleetProd,
			Version: 1,
		},
		Mailserver{
			ID:      "mail-01.do-ams3.eth.prod",
			Address: "enode://c42f368a23fa98ee546fd247220759062323249ef657d26d357a777443aec04db1b29a3a22ef3e7c548e18493ddaf51a31b0aed6079bd6ebe5ae838fcfaf3a49@178.128.142.54:443",
			Fleet:   params.FleetProd,
			Version: 1,
		},
		Mailserver{
			ID:      "mail-01.gc-us-central1-a.eth.prod",
			Address: "enode://ee2b53b0ace9692167a410514bca3024695dbf0e1a68e1dff9716da620efb195f04a4b9e873fb9b74ac84de801106c465b8e2b6c4f0d93b8749d1578bfcaf03e@104.197.238.144:443",
			Fleet:   params.FleetProd,
			Version: 1,
		},
		Mailserver{
			ID:      "mail-02.ac-cn-hongkong-c.eth.prod",
			Address: "enode://2c8de3cbb27a3d30cbb5b3e003bc722b126f5aef82e2052aaef032ca94e0c7ad219e533ba88c70585ebd802de206693255335b100307645ab5170e88620d2a81@47.244.221.14:443",
			Fleet:   params.FleetProd,
			Version: 1,
		},
		Mailserver{
			ID:      "mail-02.do-ams3.eth.prod",
			Address: "enode://7aa648d6e855950b2e3d3bf220c496e0cae4adfddef3e1e6062e6b177aec93bc6cdcf1282cb40d1656932ebfdd565729da440368d7c4da7dbd4d004b1ac02bf8@178.128.142.26:443",
			Fleet:   params.FleetProd,
			Version: 1,
		},
		Mailserver{
			ID:      "mail-02.gc-us-central1-a.eth.prod",
			Address: "enode://30211cbd81c25f07b03a0196d56e6ce4604bb13db773ff1c0ea2253547fafd6c06eae6ad3533e2ba39d59564cfbdbb5e2ce7c137a5ebb85e99dcfc7a75f99f55@23.236.58.92:443",
			Fleet:   params.FleetProd,
			Version: 1,
		},
		Mailserver{
			ID:      "mail-03.ac-cn-hongkong-c.eth.prod",
			Address: "enode://e85f1d4209f2f99da801af18db8716e584a28ad0bdc47fbdcd8f26af74dbd97fc279144680553ec7cd9092afe683ddea1e0f9fc571ebcb4b1d857c03a088853d@47.244.129.82:443",
			Fleet:   params.FleetProd,
			Version: 1,
		},
		Mailserver{
			ID:      "mail-03.do-ams3.eth.prod",
			Address: "enode://8a64b3c349a2e0ef4a32ea49609ed6eb3364be1110253c20adc17a3cebbc39a219e5d3e13b151c0eee5d8e0f9a8ba2cd026014e67b41a4ab7d1d5dd67ca27427@178.128.142.94:443",
			Fleet:   params.FleetProd,
			Version: 1,
		},
		Mailserver{
			ID:      "mail-03.gc-us-central1-a.eth.prod",
			Address: "enode://44160e22e8b42bd32a06c1532165fa9e096eebedd7fa6d6e5f8bbef0440bc4a4591fe3651be68193a7ec029021cdb496cfe1d7f9f1dc69eb99226e6f39a7a5d4@35.225.221.245:443",
			Fleet:   params.FleetProd,
			Version: 1,
		},
		Mailserver{
			ID:      "node-01.ac-cn-hongkong-c.waku.sandbox",
			Address: "/dns4/node-01.ac-cn-hongkong-c.waku.sandbox.status.im/tcp/30303/p2p/16Uiu2HAmSJvSJphxRdbnigUV5bjRRZFBhTtWFTSyiKaQByCjwmpV",
			Fleet:   params.FleetWakuSandbox,
			Version: 2,
		},
		Mailserver{
			ID:      "node-01.do-ams3.waku.sandbox",
			Address: "/dns4/node-01.do-ams3.waku.sandbox.status.im/tcp/30303/p2p/16Uiu2HAmQSMNExfUYUqfuXWkD5DaNZnMYnigRxFKbk3tcEFQeQeE",
			Fleet:   params.FleetWakuSandbox,
			Version: 2,
		},
		Mailserver{
			ID:      "node-01.gc-us-central1-a.waku.sandbox",
			Address: "/dns4/node-01.gc-us-central1-a.waku.sandbox.status.im/tcp/30303/p2p/16Uiu2HAm6fyqE1jB5MonzvoMdU8v76bWV8ZeNpncDamY1MQXfjdB",
			Fleet:   params.FleetWakuSandbox,
			Version: 2,
		},
		Mailserver{
			ID:      "node-01.ac-cn-hongkong-c.waku.test",
			Address: "/dns4/node-01.ac-cn-hongkong-c.waku.test.statusim.net/tcp/30303/p2p/16Uiu2HAkzHaTP5JsUwfR9NR8Rj9HC24puS6ocaU8wze4QrXr9iXp",
			Fleet:   params.FleetWakuTest,
			Version: 2,
		},
		Mailserver{
			ID:      "node-01.do-ams3.waku.test",
			Address: "/dns4/node-01.do-ams3.waku.test.statusim.net/tcp/30303/p2p/16Uiu2HAkykgaECHswi3YKJ5dMLbq2kPVCo89fcyTd38UcQD6ej5W",
			Fleet:   params.FleetWakuTest,
			Version: 2,
		},
		Mailserver{
			ID:      "node-01.gc-us-central1-a.waku.test",
			Address: "/dns4/node-01.gc-us-central1-a.waku.test.statusim.net/tcp/30303/p2p/16Uiu2HAmDCp8XJ9z1ev18zuv8NHekAsjNyezAvmMfFEJkiharitG",
			Fleet:   params.FleetWakuTest,
			Version: 2,
		},
		Mailserver{
			ID:      "node-01.ac-cn-hongkong-c.status.test",
			Address: "/dns4/node-01.ac-cn-hongkong-c.status.test.statusim.net/tcp/30303/p2p/16Uiu2HAm2BjXxCp1sYFJQKpLLbPbwd5juxbsYofu3TsS3auvT9Yi",
			Fleet:   params.FleetStatusTest,
			Version: 2,
		},
		Mailserver{
			ID:      "node-01.do-ams3.status.test",
			Address: "/dns4/node-01.do-ams3.status.test.statusim.net/tcp/30303/p2p/16Uiu2HAkukebeXjTQ9QDBeNDWuGfbaSg79wkkhK4vPocLgR6QFDf",
			Fleet:   params.FleetStatusTest,
			Version: 2,
		},
		Mailserver{
			ID:      "node-01.gc-us-central1-a.status.test",
			Address: "/dns4/node-01.gc-us-central1-a.status.test.statusim.net/tcp/30303/p2p/16Uiu2HAmGDX3iAFox93PupVYaHa88kULGqMpJ7AEHGwj3jbMtt76",
			Fleet:   params.FleetStatusTest,
			Version: 2,
		},
		Mailserver{
			ID:      "store-01.do-ams3.status.prod",
			Address: "/dns4/store-01.do-ams3.status.prod.statusim.net/tcp/30303/p2p/16Uiu2HAmAUdrQ3uwzuE4Gy4D56hX6uLKEeerJAnhKEHZ3DxF1EfT",
			Fleet:   params.FleetStatusProd,
			Version: 2,
		},
		Mailserver{
			ID:      "store-02.do-ams3.status.prod",
			Address: "/dns4/store-02.do-ams3.status.prod.statusim.net/tcp/30303/p2p/16Uiu2HAm9aDJPkhGxc2SFcEACTFdZ91Q5TJjp76qZEhq9iF59x7R",
			Fleet:   params.FleetStatusProd,
			Version: 2,
		},
		Mailserver{
			ID:      "store-01.gc-us-central1-a.status.prod",
			Address: "/dns4/store-01.gc-us-central1-a.status.prod.statusim.net/tcp/30303/p2p/16Uiu2HAmMELCo218hncCtTvC2Dwbej3rbyHQcR8erXNnKGei7WPZ",
			Fleet:   params.FleetStatusProd,
			Version: 2,
		},
		Mailserver{
			ID:      "store-02.gc-us-central1-a.status.prod",
			Address: "/dns4/store-02.gc-us-central1-a.status.prod.statusim.net/tcp/30303/p2p/16Uiu2HAmJnVR7ZzFaYvciPVafUXuYGLHPzSUigqAmeNw9nJUVGeM",
			Fleet:   params.FleetStatusProd,
			Version: 2,
		},
		Mailserver{
			ID:      "store-01.ac-cn-hongkong-c.status.prod",
			Address: "/dns4/store-01.ac-cn-hongkong-c.status.prod.statusim.net/tcp/30303/p2p/16Uiu2HAm2M7xs7cLPc3jamawkEqbr7cUJX11uvY7LxQ6WFUdUKUT",
			Fleet:   params.FleetStatusProd,
			Version: 2,
		},
		Mailserver{
			ID:      "store-02.ac-cn-hongkong-c.status.prod",
			Address: "/dns4/store-02.ac-cn-hongkong-c.status.prod.statusim.net/tcp/30303/p2p/16Uiu2HAm9CQhsuwPR54q27kNj9iaQVfyRzTGKrhFmr94oD8ujU6P",
			Fleet:   params.FleetStatusProd,
			Version: 2,
		},
		Mailserver{
			ID:      "store-01.do-ams3.status.staging.status.im",
			Address: "/dns4/store-01.do-ams3.status.staging.status.im/tcp/30303/p2p/16Uiu2HAm3xVDaz6SRJ6kErwC21zBJEZjavVXg7VSkoWzaV1aMA3F",
			Fleet:   params.FleetStatusStaging,
			Version: 2,
		},
		Mailserver{
			ID:      "store-02.do-ams3.status.staging.status.im",
			Address: "/dns4/store-02.do-ams3.status.staging.status.im/tcp/30303/p2p/16Uiu2HAmCDSnT8oNpMR9HH6uipD71KstYuDCAQGpek9XDAVmqdEr",
			Fleet:   params.FleetStatusStaging,
			Version: 2,
		},
		Mailserver{
			ID:      "store-01.gc-us-central1-a.status.staging.status.im",
			Address: "/dns4/store-01.gc-us-central1-a.status.staging.status.im/tcp/30303/p2p/16Uiu2HAmB7Ur9HQqo3cWDPovRQjo57fxWWDaQx27WxSzDGhN4JKg",
			Fleet:   params.FleetStatusStaging,
			Version: 2,
		},
		Mailserver{
			ID:      "store-02.gc-us-central1-a.status.staging.status.im",
			Address: "/dns4/store-02.gc-us-central1-a.status.staging.status.im/tcp/30303/p2p/16Uiu2HAmKBd6crqQNZ6nKCSCpHCAwUPn3DUDmkcPSWUTyVXpxKsW",
			Fleet:   params.FleetStatusStaging,
			Version: 2,
		},
		Mailserver{
			ID:      "store-01.ac-cn-hongkong-c.status.staging.status.im",
			Address: "/dns4/store-01.ac-cn-hongkong-c.status.staging.status.im/tcp/30303/p2p/16Uiu2HAmMU7Y29oL6DmoJfBFv8J4JhYzYgazPL7nGKJFBV3qcj2E",
			Fleet:   params.FleetStatusStaging,
			Version: 2,
		},
		Mailserver{
			ID:      "store-02.ac-cn-hongkong-c.status.staging.status.im",
			Address: "/dns4/store-02.ac-cn-hongkong-c.status.staging.status.im/tcp/30303/p2p/16Uiu2HAmU7xtcwytXpGpeDrfyhJkiFvTkQbLB9upL5MXPLGceG9K",
			Fleet:   params.FleetStatusStaging,
			Version: 2,
		},
	}
}
