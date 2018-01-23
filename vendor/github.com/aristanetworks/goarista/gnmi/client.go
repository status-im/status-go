// Copyright (C) 2017  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package gnmi

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	gnmipb "github.com/openconfig/reference/rpc/gnmi"
)

// Config is the gnmi.Client config
type Config struct {
	Addr     string
	CAFile   string
	CertFile string
	KeyFile  string
	Password string
	Username string
	TLS      bool
}

// Dial connects to a gnmi service and returns a client
func Dial(cfg Config) gnmipb.GNMIClient {
	var opts []grpc.DialOption
	if cfg.TLS || cfg.CAFile != "" || cfg.CertFile != "" {
		tlsConfig := &tls.Config{}
		if cfg.CAFile != "" {
			b, err := ioutil.ReadFile(cfg.CAFile)
			if err != nil {
				log.Fatal(err)
			}
			cp := x509.NewCertPool()
			if !cp.AppendCertsFromPEM(b) {
				log.Fatalf("credentials: failed to append certificates")
			}
			tlsConfig.RootCAs = cp
		} else {
			tlsConfig.InsecureSkipVerify = true
		}
		if cfg.CertFile != "" {
			if cfg.KeyFile == "" {
				log.Fatalf("Please provide both -certfile and -keyfile")
			}
			cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
			if err != nil {
				log.Fatal(err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	conn, err := grpc.Dial(cfg.Addr, opts...)
	if err != nil {
		log.Fatalf("Failed to dial: %s", err)
	}

	return gnmipb.NewGNMIClient(conn)
}

// NewContext returns a new context with username and password
// metadata if they are set in cfg.
func NewContext(ctx context.Context, cfg Config) context.Context {
	if cfg.Username != "" {
		ctx = metadata.NewContext(ctx, metadata.Pairs(
			"username", cfg.Username,
			"password", cfg.Password))
	}
	return ctx
}

// NewGetRequest returns a GetRequest for the given paths
func NewGetRequest(paths [][]string) *gnmipb.GetRequest {
	req := &gnmipb.GetRequest{
		Path: make([]*gnmipb.Path, len(paths)),
	}
	for i, p := range paths {
		req.Path[i] = &gnmipb.Path{Element: p}
	}
	return req
}

// NewSubscribeRequest returns a SubscribeRequest for the given paths
func NewSubscribeRequest(paths [][]string) *gnmipb.SubscribeRequest {
	subList := &gnmipb.SubscriptionList{
		Subscription: make([]*gnmipb.Subscription, len(paths)),
	}
	for i, p := range paths {
		subList.Subscription[i] = &gnmipb.Subscription{Path: &gnmipb.Path{Element: p}}
	}
	return &gnmipb.SubscribeRequest{
		Request: &gnmipb.SubscribeRequest_Subscribe{Subscribe: subList}}
}
