// Copyright (C) 2016  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

// Package client provides helper functions for OpenConfig CLI tools.
package client

import (
	"io"
	"strings"
	"sync"

	"github.com/aristanetworks/glog"
	"github.com/golang/protobuf/proto"
	"github.com/openconfig/reference/rpc/openconfig"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const defaultPort = "6042"

// PublishFunc is the method to publish responses
type PublishFunc func(addr string, message proto.Message)

// Client is a connected gRPC client
type Client struct {
	client openconfig.OpenConfigClient
	ctx    context.Context
	device string
}

// New creates a new gRPC client and connects it
func New(username, password, addr string, opts []grpc.DialOption) *Client {
	device := addr
	if !strings.ContainsRune(addr, ':') {
		addr += ":" + defaultPort
	}
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		glog.Fatalf("Failed to dial: %s", err)
	}
	glog.Infof("Connected to %s", addr)
	client := openconfig.NewOpenConfigClient(conn)

	ctx := context.Background()
	if username != "" {
		ctx = metadata.NewContext(ctx, metadata.Pairs(
			"username", username,
			"password", password))
	}
	return &Client{
		client: client,
		device: device,
		ctx:    ctx,
	}
}

// Get sends a get request and returns the responses
func (c *Client) Get(path string) []*openconfig.Notification {
	req := &openconfig.GetRequest{
		Path: []*openconfig.Path{
			{
				Element: strings.Split(path, "/"),
			},
		},
	}
	response, err := c.client.Get(c.ctx, req)
	if err != nil {
		glog.Fatalf("Get failed: %s", err)
	}
	return response.Notification
}

// Subscribe sends subscriptions, and consumes responses.
// The given publish function is used to publish SubscribeResponses received
// for the given subscriptions, when connected to the given host, with the
// given user/pass pair, or the client-side cert specified in the gRPC opts.
// This function does not normally return so it should probably be run in its
// own goroutine.  When this function returns, the given WaitGroup is marked
// as done.
func (c *Client) Subscribe(wg *sync.WaitGroup, subscriptions []string,
	publish PublishFunc) {
	defer wg.Done()
	stream, err := c.client.Subscribe(c.ctx)
	if err != nil {
		glog.Fatalf("Subscribe failed: %s", err)
	}
	defer stream.CloseSend()

	for _, path := range subscriptions {
		sub := &openconfig.SubscribeRequest{
			Request: &openconfig.SubscribeRequest_Subscribe{
				Subscribe: &openconfig.SubscriptionList{
					Subscription: []*openconfig.Subscription{
						&openconfig.Subscription{
							Path: &openconfig.Path{Element: strings.Split(path, "/")},
						},
					},
				},
			},
		}

		glog.Infof("Sending subscribe request: %s", sub)
		err = stream.Send(sub)
		if err != nil {
			glog.Fatalf("Failed to subscribe: %s", err)
		}
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			if err != io.EOF {
				glog.Fatalf("Error received from the server: %s", err)
			}
			return
		}
		switch resp := resp.Response.(type) {
		case *openconfig.SubscribeResponse_SyncResponse:
			if !resp.SyncResponse {
				panic("initial sync failed," +
					" check that you're using a client compatible with the server")
			}
		}
		glog.V(3).Info(resp)
		publish(c.device, resp)
	}
}
