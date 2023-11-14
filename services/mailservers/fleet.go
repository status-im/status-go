package mailservers

import "github.com/status-im/status-go/params"

func DefaultMailservers() []Mailserver {
	return []Mailserver{
		Mailserver{
			ID:      "16Uiu2HAmVqaWAdzBCtXC92iT4xNVFkto8MfRgMHqDCEmqnKPYWM3",
			Address: "/ip4/192.168.1.188/tcp/60001/p2p/16Uiu2HAmVqaWAdzBCtXC92iT4xNVFkto8MfRgMHqDCEmqnKPYWM3",
			Fleet:   params.FleetStatusProd,
			Version: 2,
		},Mailserver{
			ID:      "16Uiu2HAmUV8TtQJ4L5qoSccNEpAWy6WwauGnxNc9PHMaxnkuVS7W",
			Address: "/ip4/192.168.1.188/tcp/60002/p2p/16Uiu2HAmUV8TtQJ4L5qoSccNEpAWy6WwauGnxNc9PHMaxnkuVS7W",
			Fleet:   params.FleetStatusProd,
			Version: 2,
		},Mailserver{
			ID:      "16Uiu2HAm6XD2Axn5QWyjDHJjSe4htHgVF6tC8Jzy26ySbEJdj8fb",
			Address: "/ip4/192.168.1.188/tcp/60003/p2p/16Uiu2HAm6XD2Axn5QWyjDHJjSe4htHgVF6tC8Jzy26ySbEJdj8fb",
			Fleet:   params.FleetStatusProd,
			Version: 2,
		},
	}
}
