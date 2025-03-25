package params

import (
	wakutypes "github.com/status-im/status-go/waku/types"
)

// Define available fleets.
const (
	FleetUndefined     = ""
	FleetProd          = "eth.prod"
	FleetStatusStaging = "status.staging"
	FleetStatusProd    = "status.prod"
	FleetWakuSandbox   = "waku.sandbox"
	FleetWakuTest      = "waku.test"
)

type FleetInfo struct {
	WakuNodes            []string               `json:"wakuNodes"`
	DiscV5BootstrapNodes []string               `json:"discV5BootstrapNodes"`
	ClusterID            uint16                 `json:"clusterID"`
	StoreNodes           []wakutypes.Mailserver `json:"storeNodes"`
}
type FleetsMap map[string]FleetInfo

// DefaultWakuNodes is a list of "supported" fleets. This list is populated to clients UI settings.
var supportedFleets = FleetsMap{
	FleetStatusStaging: {
		ClusterID: 16,
		WakuNodes: []string{
			"enrtree://AI4W5N5IFEUIHF5LESUAOSMV6TKWF2MB6GU2YK7PU4TYUGUNOCEPW@boot.staging.status.nodes.status.im",
		},
		DiscV5BootstrapNodes: []string{
			"enrtree://AI4W5N5IFEUIHF5LESUAOSMV6TKWF2MB6GU2YK7PU4TYUGUNOCEPW@boot.staging.status.nodes.status.im",
			"enr:-QEQuEBuiQgFlJNcv255042zwyl4pOBOivakX8N30Dr9vaaEU2q8-7N4GVY4Hk87iEKELjlIXTpE9Wj6EQq1lrBuc7ayAYJpZIJ2NIJpcISPxvrpim11bHRpYWRkcnO4YAAtNihib290LTAxLmRvLWFtczMuc3RhdHVzLnN0YWdpbmcuc3RhdHVzLmltBnZfAC82KGJvb3QtMDEuZG8tYW1zMy5zdGF0dXMuc3RhZ2luZy5zdGF0dXMuaW0GAbveA4Jyc40AEAUAAQAgAEAAgAEAiXNlY3AyNTZrMaEDq-yGgpuoUG6NKkbIDRmrMiT-bEVzFlpWLEK_rF3yKUaDdGNwgnZfg3VkcIIjKIV3YWt1Mg0",
			"enr:-QEiuED2UusuHo1d6WN2-tHjtj0T0gdnsOh7aRZnFF6OEYLDbyxOtQo2_4dFUHhc9xm5SHNrWJJq8X7FRsxc4VCMGjjbAYJpZIJ2NIJpcIRoxQVgim11bHRpYWRkcnO4cgA2NjFib290LTAxLmdjLXVzLWNlbnRyYWwxLWEuc3RhdHVzLnN0YWdpbmcuc3RhdHVzLmltBnZfADg2MWJvb3QtMDEuZ2MtdXMtY2VudHJhbDEtYS5zdGF0dXMuc3RhZ2luZy5zdGF0dXMuaW0GAbveA4Jyc40AEAUAAQAgAEAAgAEAiXNlY3AyNTZrMaEDNAvlGjekD1YV4WpmjwArGAH2g9kHFJnMRfgUhcIkoA2DdGNwgnZfg3VkcIIjKIV3YWt1Mg0",
			"enr:-QEiuECJPv2vL00Jp5sTEMAFyW7qXkK2cFgphlU_G8-FJuJqoW_D5aWIy3ylGdv2K8DkiG7PWgng4Ql_VI7Qc2RhBdwfAYJpZIJ2NIJpcIQvTKi6im11bHRpYWRkcnO4cgA2NjFib290LTAxLmFjLWNuLWhvbmdrb25nLWMuc3RhdHVzLnN0YWdpbmcuc3RhdHVzLmltBnZfADg2MWJvb3QtMDEuYWMtY24taG9uZ2tvbmctYy5zdGF0dXMuc3RhZ2luZy5zdGF0dXMuaW0GAbveA4Jyc40AEAUAAQAgAEAAgAEAiXNlY3AyNTZrMaEDkbgV7oqPNmFtX5FzSPi9WH8kkmrPB1R3n9xRXge91M-DdGNwgnZfg3VkcIIjKIV3YWt1Mg0",
		},
		StoreNodes: []wakutypes.Mailserver{
			{
				ID:    "store-01.do-ams3.status.staging.status.im",
				ENR:   wakutypes.MustDecodeENR("enr:-QESuECcvLR_0SfeYbcXqxmQrnQwtdhDd4DlqzpYAsmCiWOJAkRBhXFXBNS99tzi53QrECSw9UyOhazKb7memK8eMshbAYJpZIJ2NIJpcIQYkE53im11bHRpYWRkcnO4YgAuNilzdG9yZS0wMS5kby1hbXMzLnN0YXR1cy5zdGFnaW5nLnN0YXR1cy5pbQZ2XwAwNilzdG9yZS0wMS5kby1hbXMzLnN0YXR1cy5zdGFnaW5nLnN0YXR1cy5pbQYBu94DgnJzjQAQBQABACAAQACAAQCJc2VjcDI1NmsxoQJ-wlTnBcknPNUG72hag4NXSa6SeozscHKtYg1Ss3pldoN0Y3CCdl-DdWRwgiMohXdha3UyAw"),
				Addr:  wakutypes.MustDecodeMultiaddress("/dns4/store-01.do-ams3.status.staging.status.im/tcp/30303/p2p/16Uiu2HAm3xVDaz6SRJ6kErwC21zBJEZjavVXg7VSkoWzaV1aMA3F"),
				Fleet: FleetStatusStaging,
			},
			{
				ID:    "store-02.do-ams3.status.staging.status.im",
				ENR:   wakutypes.MustDecodeENR("enr:-QESuEDD651gYmOSqKbT-wmVzMmgQBpEsoqm6JdLgX-xqPo6PGKasYBooHujyVVR9Q_G3XY1LlnOsSgcelvs4vfdumB8AYJpZIJ2NIJpcIQYkE54im11bHRpYWRkcnO4YgAuNilzdG9yZS0wMi5kby1hbXMzLnN0YXR1cy5zdGFnaW5nLnN0YXR1cy5pbQZ2XwAwNilzdG9yZS0wMi5kby1hbXMzLnN0YXR1cy5zdGFnaW5nLnN0YXR1cy5pbQYBu94DgnJzjQAQBQABACAAQACAAQCJc2VjcDI1NmsxoQL5dMmr5GzH0Fton8NGBlUW_rZG8-f3Ph0XhvMUMeVIM4N0Y3CCdl-DdWRwgiMohXdha3UyAw"),
				Addr:  wakutypes.MustDecodeMultiaddress("/dns4/store-02.do-ams3.status.staging.status.im/tcp/30303/p2p/16Uiu2HAmCDSnT8oNpMR9HH6uipD71KstYuDCAQGpek9XDAVmqdEr"),
				Fleet: FleetStatusStaging,
			},
			{
				ID:    "store-01.gc-us-central1-a.status.staging.status.im",
				ENR:   wakutypes.MustDecodeENR("enr:-QEkuEByZrFPBtvSWe0YjNrpupQzQg5nyJsQuiTVjLX8V_Du2lcFWg2GIMBWvLR7kCiwQtxgNCPH_lxXMxVbEkovBdQOAYJpZIJ2NIJpcIQj4OfRim11bHRpYWRkcnO4dAA3NjJzdG9yZS0wMS5nYy11cy1jZW50cmFsMS1hLnN0YXR1cy5zdGFnaW5nLnN0YXR1cy5pbQZ2XwA5NjJzdG9yZS0wMS5nYy11cy1jZW50cmFsMS1hLnN0YXR1cy5zdGFnaW5nLnN0YXR1cy5pbQYBu94DgnJzjQAQBQABACAAQACAAQCJc2VjcDI1NmsxoQLpEfMK4rQu4Vj5p2mH3YpiNCaiB8Q9JWuCa5sHA1BoJ4N0Y3CCdl-DdWRwgiMohXdha3UyAw"),
				Addr:  wakutypes.MustDecodeMultiaddress("/dns4/store-01.gc-us-central1-a.status.staging.status.im/tcp/30303/p2p/16Uiu2HAmB7Ur9HQqo3cWDPovRQjo57fxWWDaQx27WxSzDGhN4JKg"),
				Fleet: FleetStatusStaging,
			},
			{
				ID:    "store-02.gc-us-central1-a.status.staging.status.im",
				ENR:   wakutypes.MustDecodeENR("enr:-QEkuEAPht9zlTwD-vZWOlYXehHnrTpTMu0YaTaqHjYmyuhaM0bvLWLKjvH4df9TRDKI7dl9HM15LS3Qeqy9Vf83kfjlAYJpZIJ2NIJpcIQiSIy3im11bHRpYWRkcnO4dAA3NjJzdG9yZS0wMi5nYy11cy1jZW50cmFsMS1hLnN0YXR1cy5zdGFnaW5nLnN0YXR1cy5pbQZ2XwA5NjJzdG9yZS0wMi5nYy11cy1jZW50cmFsMS1hLnN0YXR1cy5zdGFnaW5nLnN0YXR1cy5pbQYBu94DgnJzjQAQBQABACAAQACAAQCJc2VjcDI1NmsxoQNg_xiKKSUfqa798Ay2GZzh1iRx58F7v5TQBfzFb9T0QYN0Y3CCdl-DdWRwgiMohXdha3UyAw"),
				Addr:  wakutypes.MustDecodeMultiaddress("/dns4/store-02.gc-us-central1-a.status.staging.status.im/tcp/30303/p2p/16Uiu2HAmKBd6crqQNZ6nKCSCpHCAwUPn3DUDmkcPSWUTyVXpxKsW"),
				Fleet: FleetStatusStaging,
			},
			{
				ID:    "store-01.ac-cn-hongkong-c.status.staging.status.im",
				ENR:   wakutypes.MustDecodeENR("enr:-QEkuEDCHMeQ7rxmz7TPJy87bLeYobNhxZ90Fkycawu-WlSHQ1uaqrjxLL0btJpnv4gekPoqU6RjkQJSzsS4NxU6CWnPAYJpZIJ2NIJpcIQI2s6Gim11bHRpYWRkcnO4dAA3NjJzdG9yZS0wMS5hYy1jbi1ob25na29uZy1jLnN0YXR1cy5zdGFnaW5nLnN0YXR1cy5pbQZ2XwA5NjJzdG9yZS0wMS5hYy1jbi1ob25na29uZy1jLnN0YXR1cy5zdGFnaW5nLnN0YXR1cy5pbQYBu94DgnJzjQAQBQABACAAQACAAQCJc2VjcDI1NmsxoQOC7-rlGZ1POquzYNLxqu1_RddP7HXIGafRaEKM934p54N0Y3CCdl-DdWRwgiMohXdha3UyAw"),
				Addr:  wakutypes.MustDecodeMultiaddress("/dns4/store-01.ac-cn-hongkong-c.status.staging.status.im/tcp/30303/p2p/16Uiu2HAmMU7Y29oL6DmoJfBFv8J4JhYzYgazPL7nGKJFBV3qcj2E"),
				Fleet: FleetStatusStaging,
			},
			{
				ID:    "store-02.ac-cn-hongkong-c.status.staging.status.im",
				ENR:   wakutypes.MustDecodeENR("enr:-QEkuEAxgmSmx5RJ1odC-C_bXkDCE_VXTuB49ENTlI89p9uNLVKRqrwythgiAtjFxAokR4gvHvQMcX5Ts0N70Ut_kyPJAYJpZIJ2NIJpcIQvTLKkim11bHRpYWRkcnO4dAA3NjJzdG9yZS0wMi5hYy1jbi1ob25na29uZy1jLnN0YXR1cy5zdGFnaW5nLnN0YXR1cy5pbQZ2XwA5NjJzdG9yZS0wMi5hYy1jbi1ob25na29uZy1jLnN0YXR1cy5zdGFnaW5nLnN0YXR1cy5pbQYBu94DgnJzjQAQBQABACAAQACAAQCJc2VjcDI1NmsxoQPlyFXKktjIFNaZtTIFI_4ZfNyt3RKWxSPEyH_nb7-YFoN0Y3CCdl-DdWRwgiMohXdha3UyAw"),
				Addr:  wakutypes.MustDecodeMultiaddress("/dns4/store-02.ac-cn-hongkong-c.status.staging.status.im/tcp/30303/p2p/16Uiu2HAmU7xtcwytXpGpeDrfyhJkiFvTkQbLB9upL5MXPLGceG9K"),
				Fleet: FleetStatusStaging,
			},
		},
	},
	FleetStatusProd: {
		ClusterID: 16,
		WakuNodes: []string{
			"enrtree://AMOJVZX4V6EXP7NTJPMAYJYST2QP6AJXYW76IU6VGJS7UVSNDYZG4@boot.prod.status.nodes.status.im",
		},
		DiscV5BootstrapNodes: []string{
			"enrtree://AMOJVZX4V6EXP7NTJPMAYJYST2QP6AJXYW76IU6VGJS7UVSNDYZG4@boot.prod.status.nodes.status.im",
			"enr:-QEKuED9AJm2HGgrRpVaJY2nj68ao_QiPeUT43sK-aRM7sMJ6R4G11OSDOwnvVacgN1sTw-K7soC5dzHDFZgZkHU0u-XAYJpZIJ2NIJpcISnYxMvim11bHRpYWRkcnO4WgAqNiVib290LTAxLmRvLWFtczMuc3RhdHVzLnByb2Quc3RhdHVzLmltBnZfACw2JWJvb3QtMDEuZG8tYW1zMy5zdGF0dXMucHJvZC5zdGF0dXMuaW0GAbveA4Jyc40AEAUAAQAgAEAAgAEAiXNlY3AyNTZrMaEC3rRtFQSgc24uWewzXaxTY8hDAHB8sgnxr9k8Rjb5GeSDdGNwgnZfg3VkcIIjKIV3YWt1Mg0",
			"enr:-QEcuED7ww5vo2rKc1pyBp7fubBUH-8STHEZHo7InjVjLblEVyDGkjdTI9VdqmYQOn95vuQH-Htku17WSTzEufx-Wg4mAYJpZIJ2NIJpcIQihw1Xim11bHRpYWRkcnO4bAAzNi5ib290LTAxLmdjLXVzLWNlbnRyYWwxLWEuc3RhdHVzLnByb2Quc3RhdHVzLmltBnZfADU2LmJvb3QtMDEuZ2MtdXMtY2VudHJhbDEtYS5zdGF0dXMucHJvZC5zdGF0dXMuaW0GAbveA4Jyc40AEAUAAQAgAEAAgAEAiXNlY3AyNTZrMaECxjqgDQ0WyRSOilYU32DA5k_XNlDis3m1VdXkK9xM6kODdGNwgnZfg3VkcIIjKIV3YWt1Mg0",
			"enr:-QEcuEAoShWGyN66wwusE3Ri8hXBaIkoHZHybUB8cCPv5v3ypEf9OCg4cfslJxZFANl90s-jmMOugLUyBx4EfOBNJ6_VAYJpZIJ2NIJpcIQI2hdMim11bHRpYWRkcnO4bAAzNi5ib290LTAxLmFjLWNuLWhvbmdrb25nLWMuc3RhdHVzLnByb2Quc3RhdHVzLmltBnZfADU2LmJvb3QtMDEuYWMtY24taG9uZ2tvbmctYy5zdGF0dXMucHJvZC5zdGF0dXMuaW0GAbveA4Jyc40AEAUAAQAgAEAAgAEAiXNlY3AyNTZrMaEDP7CbRk-YKJwOFFM4Z9ney0GPc7WPJaCwGkpNRyla7mCDdGNwgnZfg3VkcIIjKIV3YWt1Mg0",
		},
		StoreNodes: []wakutypes.Mailserver{
			{
				ID:    "store-01.do-ams3.status.prod",
				ENR:   wakutypes.MustDecodeENR("enr:-QEMuEAs8JmmyUI3b9v_ADqYtELHUYAsAMS21lA2BMtrzF86tVmyy9cCrhmzfHGHx_g3nybn7jIRybzXTGNj3C2KzrriAYJpZIJ2NIJpcISf3_Jeim11bHRpYWRkcnO4XAArNiZzdG9yZS0wMS5kby1hbXMzLnN0YXR1cy5wcm9kLnN0YXR1cy5pbQZ2XwAtNiZzdG9yZS0wMS5kby1hbXMzLnN0YXR1cy5wcm9kLnN0YXR1cy5pbQYBu94DgnJzjQAQBQABACAAQACAAQCJc2VjcDI1NmsxoQLfoaQH3oSYW59yxEBfeAZbltmUnC4BzYkHqer2VQMTyoN0Y3CCdl-DdWRwgiMohXdha3UyAw"),
				Addr:  wakutypes.MustDecodeMultiaddress("/dns4/store-01.do-ams3.status.prod.status.im/tcp/30303/p2p/16Uiu2HAmAUdrQ3uwzuE4Gy4D56hX6uLKEeerJAnhKEHZ3DxF1EfT"),
				Fleet: FleetStatusProd,
			},
			{
				ID:    "store-02.do-ams3.status.prod",
				ENR:   wakutypes.MustDecodeENR("enr:-QEMuEDuTfD47Hz_NXDwf7LJMf0qhjp3CQhZ9Fy0Ulp4XehtEzewBzmJCoe77hjno3khH8kX2B9B1DgbJuc2n32fMZvOAYJpZIJ2NIJpcISf3_Kaim11bHRpYWRkcnO4XAArNiZzdG9yZS0wMi5kby1hbXMzLnN0YXR1cy5wcm9kLnN0YXR1cy5pbQZ2XwAtNiZzdG9yZS0wMi5kby1hbXMzLnN0YXR1cy5wcm9kLnN0YXR1cy5pbQYBu94DgnJzjQAQBQABACAAQACAAQCJc2VjcDI1NmsxoQLSM62HmqGpZ382YM4CyI-MCIlkxMP7ZbOwqwRPvk9wsIN0Y3CCdl-DdWRwgiMohXdha3UyAw"),
				Addr:  wakutypes.MustDecodeMultiaddress("/dns4/store-02.do-ams3.status.prod.status.im/tcp/30303/p2p/16Uiu2HAm9aDJPkhGxc2SFcEACTFdZ91Q5TJjp76qZEhq9iF59x7R"),
				Fleet: FleetStatusProd,
			},
			{
				ID:    "store-01.gc-us-central1-a.status.prod",
				ENR:   wakutypes.MustDecodeENR("enr:-QEeuEA08-NJJDuKh6V8739MPl2G7ykaC0EWyUg21KtjQ1UtKxuE2qNy5uES2_bobr7sC5C4sS_-GhDVYMpOrM2IFc8KAYJpZIJ2NIJpcIQiqsAnim11bHRpYWRkcnO4bgA0Ni9zdG9yZS0wMS5nYy11cy1jZW50cmFsMS1hLnN0YXR1cy5wcm9kLnN0YXR1cy5pbQZ2XwA2Ni9zdG9yZS0wMS5nYy11cy1jZW50cmFsMS1hLnN0YXR1cy5wcm9kLnN0YXR1cy5pbQYBu94DgnJzjQAQBQABACAAQACAAQCJc2VjcDI1NmsxoQN_aBxNsOBrceDLyC75vBFRuzv_tWfaHG50Jc9DQztwkIN0Y3CCdl-DdWRwgiMohXdha3UyAw"),
				Addr:  wakutypes.MustDecodeMultiaddress("/dns4/store-01.gc-us-central1-a.status.prod.status.im/tcp/30303/p2p/16Uiu2HAmMELCo218hncCtTvC2Dwbej3rbyHQcR8erXNnKGei7WPZ"),
				Fleet: FleetStatusProd,
			},
			{
				ID:    "store-02.gc-us-central1-a.status.prod",
				ENR:   wakutypes.MustDecodeENR("enr:-QEeuECQiv4VvUk04UnU3wxKXgWvErYcGMgYU8aDuc8VvEt1km2GvcEBq-R9XT-loNL5PZjxGKzB1rDtCOQaFVYQtgPnAYJpZIJ2NIJpcIQiqpoCim11bHRpYWRkcnO4bgA0Ni9zdG9yZS0wMi5nYy11cy1jZW50cmFsMS1hLnN0YXR1cy5wcm9kLnN0YXR1cy5pbQZ2XwA2Ni9zdG9yZS0wMi5nYy11cy1jZW50cmFsMS1hLnN0YXR1cy5wcm9kLnN0YXR1cy5pbQYBu94DgnJzjQAQBQABACAAQACAAQCJc2VjcDI1NmsxoQNbEg1bkMJCBiD5Tje3Z_11R-kd9munZF0v4iiYZa1jgoN0Y3CCdl-DdWRwgiMohXdha3UyAw"),
				Addr:  wakutypes.MustDecodeMultiaddress("/dns4/store-02.gc-us-central1-a.status.prod.status.im/tcp/30303/p2p/16Uiu2HAmJnVR7ZzFaYvciPVafUXuYGLHPzSUigqAmeNw9nJUVGeM"),
				Fleet: FleetStatusProd,
			},
			{
				ID:    "store-01.ac-cn-hongkong-c.status.prod",
				ENR:   wakutypes.MustDecodeENR("enr:-QEeuED6hfo5OQICpfwrjuG-qC8MMjw8bsLrF-xi8tY4nz3h7nl_KOXA2C1q7gXOzJ-bROP2ZzITdRlP0HN57jiBuim9AYJpZIJ2NIJpcIQI2kpJim11bHRpYWRkcnO4bgA0Ni9zdG9yZS0wMS5hYy1jbi1ob25na29uZy1jLnN0YXR1cy5wcm9kLnN0YXR1cy5pbQZ2XwA2Ni9zdG9yZS0wMS5hYy1jbi1ob25na29uZy1jLnN0YXR1cy5wcm9kLnN0YXR1cy5pbQYBu94DgnJzjQAQBQABACAAQACAAQCJc2VjcDI1NmsxoQJm10jdarzx9hcdhRKGfsAyS0Hc5pWj3yhyTvT5FIwKGIN0Y3CCdl-DdWRwgiMohXdha3UyAw"),
				Addr:  wakutypes.MustDecodeMultiaddress("/dns4/store-01.ac-cn-hongkong-c.status.prod.status.im/tcp/30303/p2p/16Uiu2HAm2M7xs7cLPc3jamawkEqbr7cUJX11uvY7LxQ6WFUdUKUT"),
				Fleet: FleetStatusProd,
			},
			{
				ID:    "store-02.ac-cn-hongkong-c.status.prod",
				ENR:   wakutypes.MustDecodeENR("enr:-QEeuEC0VBi0VMXNL4oQUfdAJL7RBXpWyB54TqUDt93Li3yuax4ohwMMIAmI6sg2jgH_HxgDRy5Ar-5CbMDW1EFxYFplAYJpZIJ2NIJpcIQI2nnoim11bHRpYWRkcnO4bgA0Ni9zdG9yZS0wMi5hYy1jbi1ob25na29uZy1jLnN0YXR1cy5wcm9kLnN0YXR1cy5pbQZ2XwA2Ni9zdG9yZS0wMi5hYy1jbi1ob25na29uZy1jLnN0YXR1cy5wcm9kLnN0YXR1cy5pbQYBu94DgnJzjQAQBQABACAAQACAAQCJc2VjcDI1NmsxoQLMncuu6pJ3DQRzYUqkB1PbaRxZXIGJi8waKbbBFbOSNIN0Y3CCdl-DdWRwgiMohXdha3UyAw"),
				Addr:  wakutypes.MustDecodeMultiaddress("/dns4/store-02.ac-cn-hongkong-c.status.prod.status.im/tcp/30303/p2p/16Uiu2HAm9CQhsuwPR54q27kNj9iaQVfyRzTGKrhFmr94oD8ujU6P"),
				Fleet: FleetStatusProd,
			},
		},
	},
	FleetWakuSandbox: {
		WakuNodes: []string{
			"enrtree://AIRVQ5DDA4FFWLRBCHJWUWOO6X6S4ZTZ5B667LQ6AJU6PEYDLRD5O@sandbox.waku.nodes.status.im",
		},
		DiscV5BootstrapNodes: []string{
			"enrtree://AIRVQ5DDA4FFWLRBCHJWUWOO6X6S4ZTZ5B667LQ6AJU6PEYDLRD5O@sandbox.waku.nodes.status.im",
		},
		StoreNodes: []wakutypes.Mailserver{
			{
				ID:    "node-01.ac-cn-hongkong-c.waku.sandbox",
				ENR:   wakutypes.MustDecodeENR("enr:-QEkuEBfEzJm_kigJ2HoSS_RBFJYhKHocGdkhhBr6jSUAWjLdFPp6Pj1l4yiTQp7TGHyu1kC6FyaU573VN8klLsEm-XuAYJpZIJ2NIJpcIQI2SVcim11bHRpYWRkcnO4bgA0Ni9ub2RlLTAxLmFjLWNuLWhvbmdrb25nLWMud2FrdS5zYW5kYm94LnN0YXR1cy5pbQZ2XwA2Ni9ub2RlLTAxLmFjLWNuLWhvbmdrb25nLWMud2FrdS5zYW5kYm94LnN0YXR1cy5pbQYfQN4DgnJzkwABCAAAAAEAAgADAAQABQAGAAeJc2VjcDI1NmsxoQOwsS69tgD7u1K50r5-qG5hweuTwa0W26aYPnvivpNlrYN0Y3CCdl-DdWRwgiMohXdha3UyDw"),
				Addr:  wakutypes.MustDecodeMultiaddress("/dns4/node-01.ac-cn-hongkong-c.waku.sandbox.status.im/tcp/30303/p2p/16Uiu2HAmSJvSJphxRdbnigUV5bjRRZFBhTtWFTSyiKaQByCjwmpV"),
				Fleet: FleetWakuSandbox,
			},
			{
				ID:    "node-01.do-ams3.waku.sandbox",
				ENR:   wakutypes.MustDecodeENR("enr:-QESuEB4Dchgjn7gfAvwB00CxTA-nGiyk-aALI-H4dYSZD3rUk7bZHmP8d2U6xDiQ2vZffpo45Jp7zKNdnwDUx6g4o6XAYJpZIJ2NIJpcIRA4VDAim11bHRpYWRkcnO4XAArNiZub2RlLTAxLmRvLWFtczMud2FrdS5zYW5kYm94LnN0YXR1cy5pbQZ2XwAtNiZub2RlLTAxLmRvLWFtczMud2FrdS5zYW5kYm94LnN0YXR1cy5pbQYfQN4DgnJzkwABCAAAAAEAAgADAAQABQAGAAeJc2VjcDI1NmsxoQOvD3S3jUNICsrOILlmhENiWAMmMVlAl6-Q8wRB7hidY4N0Y3CCdl-DdWRwgiMohXdha3UyDw"),
				Addr:  wakutypes.MustDecodeMultiaddress("/dns4/node-01.do-ams3.waku.sandbox.status.im/tcp/30303/p2p/16Uiu2HAmQSMNExfUYUqfuXWkD5DaNZnMYnigRxFKbk3tcEFQeQeE"),
				Fleet: FleetWakuSandbox,
			},
			{
				ID:    "node-01.gc-us-central1-a.waku.sandbox",
				ENR:   wakutypes.MustDecodeENR("enr:-QEkuEBIkb8q8_mrorHndoXH9t5N6ZfD-jehQCrYeoJDPHqT0l0wyaONa2-piRQsi3oVKAzDShDVeoQhy0uwN1xbZfPZAYJpZIJ2NIJpcIQiQlleim11bHRpYWRkcnO4bgA0Ni9ub2RlLTAxLmdjLXVzLWNlbnRyYWwxLWEud2FrdS5zYW5kYm94LnN0YXR1cy5pbQZ2XwA2Ni9ub2RlLTAxLmdjLXVzLWNlbnRyYWwxLWEud2FrdS5zYW5kYm94LnN0YXR1cy5pbQYfQN4DgnJzkwABCAAAAAEAAgADAAQABQAGAAeJc2VjcDI1NmsxoQKnGt-GSgqPSf3IAPM7bFgTlpczpMZZLF3geeoNNsxzSoN0Y3CCdl-DdWRwgiMohXdha3UyDw"),
				Addr:  wakutypes.MustDecodeMultiaddress("/dns4/node-01.gc-us-central1-a.waku.sandbox.status.im/tcp/30303/p2p/16Uiu2HAm6fyqE1jB5MonzvoMdU8v76bWV8ZeNpncDamY1MQXfjdB"),
				Fleet: FleetWakuSandbox,
			},
		},
	},
	FleetWakuTest: {
		WakuNodes: []string{
			"enrtree://AOGYWMBYOUIMOENHXCHILPKY3ZRFEULMFI4DOM442QSZ73TT2A7VI@test.waku.nodes.status.im",
		},
		DiscV5BootstrapNodes: []string{
			"enrtree://AOGYWMBYOUIMOENHXCHILPKY3ZRFEULMFI4DOM442QSZ73TT2A7VI@test.waku.nodes.status.im",
		},
		StoreNodes: []wakutypes.Mailserver{
			{
				ID:    "node-01.ac-cn-hongkong-c.waku.test",
				ENR:   wakutypes.MustDecodeENR("enr:-QEeuECvvBe6kIzHgMv_mD1YWQ3yfOfid2MO9a_A6ZZmS7E0FmAfntz2ZixAnPXvLWDJ81ARp4oV9UM4WXyc5D5USdEPAYJpZIJ2NIJpcIQI2ttrim11bHRpYWRkcnO4aAAxNixub2RlLTAxLmFjLWNuLWhvbmdrb25nLWMud2FrdS50ZXN0LnN0YXR1cy5pbQZ2XwAzNixub2RlLTAxLmFjLWNuLWhvbmdrb25nLWMud2FrdS50ZXN0LnN0YXR1cy5pbQYfQN4DgnJzkwABCAAAAAEAAgADAAQABQAGAAeJc2VjcDI1NmsxoQJIN4qwz3v4r2Q8Bv8zZD0eqBcKw6bdLvdkV7-JLjqIj4N0Y3CCdl-DdWRwgiMohXdha3UyDw"),
				Addr:  wakutypes.MustDecodeMultiaddress("/dns4/node-01.ac-cn-hongkong-c.waku.test.statusim.net/tcp/30303/p2p/16Uiu2HAkzHaTP5JsUwfR9NR8Rj9HC24puS6ocaU8wze4QrXr9iXp"),
				Fleet: FleetWakuTest,
			},
			{
				ID:    "node-01.do-ams3.waku.test",
				ENR:   wakutypes.MustDecodeENR("enr:-QEMuEDbayK340kH24XzK5FPIYNzWNYuH01NASNIb1skZfe_6l4_JSsG-vZ0LgN4Cgzf455BaP5zrxMQADHL5OQpbW6OAYJpZIJ2NIJpcISygI2rim11bHRpYWRkcnO4VgAoNiNub2RlLTAxLmRvLWFtczMud2FrdS50ZXN0LnN0YXR1cy5pbQZ2XwAqNiNub2RlLTAxLmRvLWFtczMud2FrdS50ZXN0LnN0YXR1cy5pbQYfQN4DgnJzkwABCAAAAAEAAgADAAQABQAGAAeJc2VjcDI1NmsxoQJATXRSRSUyTw_QLB6H_U3oziVQgNRgrXpK7wp2AMyNxYN0Y3CCdl-DdWRwgiMohXdha3UyDw"),
				Addr:  wakutypes.MustDecodeMultiaddress("/dns4/node-01.do-ams3.waku.test.statusim.net/tcp/30303/p2p/16Uiu2HAkykgaECHswi3YKJ5dMLbq2kPVCo89fcyTd38UcQD6ej5W"),
				Fleet: FleetWakuTest,
			},
			{
				ID:    "node-01.gc-us-central1-a.waku.test",
				ENR:   wakutypes.MustDecodeENR("enr:-QEeuEBO08GSjWDOV13HTf6L7iFoPQhv4S0-_Bd7Of3lFCBNBmpB9j6pGLedkX88KAXm6BFCS4ViQ_rLeDQuzj9Q6fs9AYJpZIJ2NIJpcIQiEAFDim11bHRpYWRkcnO4aAAxNixub2RlLTAxLmdjLXVzLWNlbnRyYWwxLWEud2FrdS50ZXN0LnN0YXR1cy5pbQZ2XwAzNixub2RlLTAxLmdjLXVzLWNlbnRyYWwxLWEud2FrdS50ZXN0LnN0YXR1cy5pbQYfQN4DgnJzkwABCAAAAAEAAgADAAQABQAGAAeJc2VjcDI1NmsxoQMIJwesBVgUiBCi8yiXGx7RWylBQkYm1U9dvEy-neLG2YN0Y3CCdl-DdWRwgiMohXdha3UyDw"),
				Addr:  wakutypes.MustDecodeMultiaddress("/dns4/node-01.gc-us-central1-a.waku.test.statusim.net/tcp/30303/p2p/16Uiu2HAmDCp8XJ9z1ev18zuv8NHekAsjNyezAvmMfFEJkiharitG"),
				Fleet: FleetWakuTest,
			},
		},
	},
}

var defaultPushNotificationServers = []string{
	"401ba5eda402678dc78a0a40fd0795f4ea8b1e34972c4d15cf33ac01292341c89f0cbc637fa9f7a3ffe0b9dfe90e9cdae7a14925500ab01b6a91c67bae42a97a",
	"181141b1d111908aaf05f4788e6778ec07073a1d4e1ce43c73815c40ee4e7345a1cbf5a90a45f601bf3763f12be63b01624ba1f36eeb9572455e7034b8f9f2c4",
	"5ffc34d5ffda180d94cd3974d9ed2bb082ede68f342babdbe801ceffb7da902087d43f9aa961c7b85029358874c08ef04ecad9f1d95a1f0e448cbdd5d04350c7",
}

func DefaultWakuNodes(fleet string) []string {
	return supportedFleets[fleet].WakuNodes
}

func DefaultDiscV5Nodes(fleet string) []string {
	return supportedFleets[fleet].DiscV5BootstrapNodes
}

func DefaultClusterID(fleet string) uint16 {
	return supportedFleets[fleet].ClusterID
}

func IsFleetSupported(fleet string) bool {
	_, ok := supportedFleets[fleet]
	return ok
}

func GetSupportedFleets() FleetsMap {
	return supportedFleets
}

func DefaultStoreNodes(fleet string) []wakutypes.Mailserver {
	return supportedFleets[fleet].StoreNodes
}

func DefaultPushNotificationServers() []string {
	return defaultPushNotificationServers
}

func DefaultClusterConfig(fleet string) ClusterConfig {
	return ClusterConfig{
		Enabled:                  true,
		Fleet:                    fleet,
		WakuNodes:                DefaultWakuNodes(fleet),
		DiscV5BootstrapNodes:     DefaultDiscV5Nodes(fleet),
		ClusterID:                DefaultClusterID(fleet),
		PushNotificationsServers: DefaultPushNotificationServers(),
	}
}
