package backend

import (
	"github.com/status-im/status-go/server"
)

func (b *StatusBackend) startMediaServer(address, advertizeHost string, advertizePort int) error {
	if b.mediaServer != nil {
		if err := b.mediaServer.Stop(); err != nil {
			return err
		}
	}

	opts := []server.MediaServerOption{
		server.WithMediaServerDisableTLS(false),
		server.WithMediaServerAddress(address),
		server.WithMediaServerAdvertizeAddress(advertizeHost, advertizePort),
	}
	mediaServer, err := server.NewMediaServer(nil, nil, b.multiaccountsDB, nil, opts...)
	if err != nil {
		return err
	}
	mediaServer.SetDataProviders(b.appDB, b.walletDB, b.ipfs)

	b.mediaServer = mediaServer

	if err := b.mediaServer.Start(); err != nil {
		return err
	}

	return nil
}

//
//func (n *Services) startMediaServer() error {
//	if n.mediaServer != nil {
//		if err := n.mediaServer.Stop(); err != nil {
//			return err
//		}
//	}
//
//	var opts []server.MediaServerOption
//	if n.mediaServerEnableTLS != nil {
//		opts = append(opts, server.WithMediaServerDisableTLS(!*n.mediaServerEnableTLS))
//	}
//	if n.mediaServerAddress != nil {
//		opts = append(opts, server.WithMediaServerAddress(*n.mediaServerAddress))
//	}
//	opts = append(opts, server.WithMediaServerAdvertizeAddress(n.mediaServerAdvertizeHost, n.mediaServerAdvertizePort))
//	mediaServer, err := server.NewMediaServer(nil, nil, n.multiaccountsDB, nil, opts...)
//	if err != nil {
//		return err
//	}
//
//	n.mediaServer = mediaServer
//
//	if err := n.mediaServer.Start(); err != nil {
//		return err
//	}
//
//	return nil
//}
