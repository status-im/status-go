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
		{
			ID:      "node-01.ac-cn-hongkong-c.waku.sandbox",
			Address: "/dns4/node-01.ac-cn-hongkong-c.waku.sandbox.status.im/tcp/30303/p2p/16Uiu2HAmSJvSJphxRdbnigUV5bjRRZFBhTtWFTSyiKaQByCjwmpV",
			Fleet:   params.FleetWakuSandbox,
			Version: 2,
		},
		{
			ID:      "node-01.do-ams3.waku.sandbox",
			Address: "/dns4/node-01.do-ams3.waku.sandbox.status.im/tcp/30303/p2p/16Uiu2HAmQSMNExfUYUqfuXWkD5DaNZnMYnigRxFKbk3tcEFQeQeE",
			Fleet:   params.FleetWakuSandbox,
			Version: 2,
		},
		{
			ID:      "node-01.gc-us-central1-a.waku.sandbox",
			Address: "/dns4/node-01.gc-us-central1-a.waku.sandbox.status.im/tcp/30303/p2p/16Uiu2HAm6fyqE1jB5MonzvoMdU8v76bWV8ZeNpncDamY1MQXfjdB",
			Fleet:   params.FleetWakuSandbox,
			Version: 2,
		},
		{
			ID:      "node-01.ac-cn-hongkong-c.waku.test",
			Address: "/dns4/node-01.ac-cn-hongkong-c.waku.test.statusim.net/tcp/30303/p2p/16Uiu2HAkzHaTP5JsUwfR9NR8Rj9HC24puS6ocaU8wze4QrXr9iXp",
			Fleet:   params.FleetWakuTest,
			Version: 2,
		},
		{
			ID:      "node-01.do-ams3.waku.test",
			Address: "/dns4/node-01.do-ams3.waku.test.statusim.net/tcp/30303/p2p/16Uiu2HAkykgaECHswi3YKJ5dMLbq2kPVCo89fcyTd38UcQD6ej5W",
			Fleet:   params.FleetWakuTest,
			Version: 2,
		},
		{
			ID:      "node-01.gc-us-central1-a.waku.test",
			Address: "/dns4/node-01.gc-us-central1-a.waku.test.statusim.net/tcp/30303/p2p/16Uiu2HAmDCp8XJ9z1ev18zuv8NHekAsjNyezAvmMfFEJkiharitG",
			Fleet:   params.FleetWakuTest,
			Version: 2,
		},
		{
			ID:      "store-01.do-ams3.status.prod",
			Address: "/dns4/store-01.do-ams3.status.prod.status.im/tcp/30303/p2p/16Uiu2HAmAUdrQ3uwzuE4Gy4D56hX6uLKEeerJAnhKEHZ3DxF1EfT",
			Fleet:   params.FleetStatusProd,
			Version: 2,
		},
		{
			ID:      "store-02.do-ams3.status.prod",
			Address: "/dns4/store-02.do-ams3.status.prod.status.im/tcp/30303/p2p/16Uiu2HAm9aDJPkhGxc2SFcEACTFdZ91Q5TJjp76qZEhq9iF59x7R",
			Fleet:   params.FleetStatusProd,
			Version: 2,
		},
		{
			ID:      "store-01.gc-us-central1-a.status.prod",
			Address: "/dns4/store-01.gc-us-central1-a.status.prod.status.im/tcp/30303/p2p/16Uiu2HAmMELCo218hncCtTvC2Dwbej3rbyHQcR8erXNnKGei7WPZ",
			Fleet:   params.FleetStatusProd,
			Version: 2,
		},
		{
			ID:      "store-02.gc-us-central1-a.status.prod",
			Address: "/dns4/store-02.gc-us-central1-a.status.prod.status.im/tcp/30303/p2p/16Uiu2HAmJnVR7ZzFaYvciPVafUXuYGLHPzSUigqAmeNw9nJUVGeM",
			Fleet:   params.FleetStatusProd,
			Version: 2,
		},
		{
			ID:      "store-01.ac-cn-hongkong-c.status.prod",
			Address: "/dns4/store-01.ac-cn-hongkong-c.status.prod.status.im/tcp/30303/p2p/16Uiu2HAm2M7xs7cLPc3jamawkEqbr7cUJX11uvY7LxQ6WFUdUKUT",
			Fleet:   params.FleetStatusProd,
			Version: 2,
		},
		{
			ID:      "store-02.ac-cn-hongkong-c.status.prod",
			Address: "/dns4/store-02.ac-cn-hongkong-c.status.prod.status.im/tcp/30303/p2p/16Uiu2HAm9CQhsuwPR54q27kNj9iaQVfyRzTGKrhFmr94oD8ujU6P",
			Fleet:   params.FleetStatusProd,
			Version: 2,
		},
		{
			ID:      "store-01.do-ams3.status.staging.status.im",
			Address: "/dns4/store-01.do-ams3.status.staging.status.im/tcp/30303/p2p/16Uiu2HAm3xVDaz6SRJ6kErwC21zBJEZjavVXg7VSkoWzaV1aMA3F",
			Fleet:   params.FleetStatusStaging,
			Version: 2,
		},
		{
			ID:      "store-02.do-ams3.status.staging.status.im",
			Address: "/dns4/store-02.do-ams3.status.staging.status.im/tcp/30303/p2p/16Uiu2HAmCDSnT8oNpMR9HH6uipD71KstYuDCAQGpek9XDAVmqdEr",
			Fleet:   params.FleetStatusStaging,
			Version: 2,
		},
		{
			ID:      "store-01.gc-us-central1-a.status.staging.status.im",
			Address: "/dns4/store-01.gc-us-central1-a.status.staging.status.im/tcp/30303/p2p/16Uiu2HAmB7Ur9HQqo3cWDPovRQjo57fxWWDaQx27WxSzDGhN4JKg",
			Fleet:   params.FleetStatusStaging,
			Version: 2,
		},
		{
			ID:      "store-02.gc-us-central1-a.status.staging.status.im",
			Address: "/dns4/store-02.gc-us-central1-a.status.staging.status.im/tcp/30303/p2p/16Uiu2HAmKBd6crqQNZ6nKCSCpHCAwUPn3DUDmkcPSWUTyVXpxKsW",
			Fleet:   params.FleetStatusStaging,
			Version: 2,
		},
		{
			ID:      "store-01.ac-cn-hongkong-c.status.staging.status.im",
			Address: "/dns4/store-01.ac-cn-hongkong-c.status.staging.status.im/tcp/30303/p2p/16Uiu2HAmMU7Y29oL6DmoJfBFv8J4JhYzYgazPL7nGKJFBV3qcj2E",
			Fleet:   params.FleetStatusStaging,
			Version: 2,
		},
		{
			ID:      "store-02.ac-cn-hongkong-c.status.staging.status.im",
			Address: "/dns4/store-02.ac-cn-hongkong-c.status.staging.status.im/tcp/30303/p2p/16Uiu2HAmU7xtcwytXpGpeDrfyhJkiFvTkQbLB9upL5MXPLGceG9K",
			Fleet:   params.FleetStatusStaging,
			Version: 2,
		},
	}
}
