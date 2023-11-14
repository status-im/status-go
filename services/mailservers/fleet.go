package mailservers

import "github.com/status-im/status-go/params"

func DefaultMailservers() []Mailserver {
	return []Mailserver{
		Mailserver{
			ID:      "16Uiu2HAm5tZRpbHwYJSxfdt945EbeoysizD28pPmNQCyeS2S341Q",
			Address: "/ip4/139.59.255.0/tcp/60001/p2p/16Uiu2HAm5tZRpbHwYJSxfdt945EbeoysizD28pPmNQCyeS2S341Q",
			Fleet:   params.FleetStatusProd,
			Version: 2,
		},Mailserver{
			ID:      "16Uiu2HAm9VCJHSbMt6jDE4X2zdTPDyUk9APL7aPRkUPdxN9HcyW6",
			Address: "/ip4/139.59.255.0/tcp/60002/p2p/16Uiu2HAm9VCJHSbMt6jDE4X2zdTPDyUk9APL7aPRkUPdxN9HcyW6",
			Fleet:   params.FleetStatusProd,
			Version: 2,
		},Mailserver{
			ID:      "16Uiu2HAmVr2SfipJRjcjTyskMrVsgUkbdWneBzoTwJTVMXVhXUfx",
			Address: "/ip4/139.59.255.0/tcp/60003/p2p/16Uiu2HAmVr2SfipJRjcjTyskMrVsgUkbdWneBzoTwJTVMXVhXUfx",
			Fleet:   params.FleetStatusProd,
			Version: 2,
		},
	}
}
