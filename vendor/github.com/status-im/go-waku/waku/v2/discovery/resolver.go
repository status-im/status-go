package discovery

import (
	"context"
	"net"
)

func GetResolver(ctx context.Context, nameserver string) *net.Resolver {
	if nameserver == "" {
		return net.DefaultResolver
	}

	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{}
			return d.DialContext(ctx, network, net.JoinHostPort(nameserver, "53"))
		},
	}
}
