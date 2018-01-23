// Copyright (C) 2016  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

// The occli tool is a simple client to dump in JSON or text format the
// protobufs returned by the OpenConfig gRPC interface.
package main

import (
	"flag"
	"fmt"
	"sync"

	"github.com/aristanetworks/glog"
	"github.com/aristanetworks/goarista/openconfig"
	"github.com/aristanetworks/goarista/openconfig/client"
	"github.com/golang/protobuf/proto"
	pb "github.com/openconfig/reference/rpc/openconfig"
)

var getFlag = flag.String("get", "",
	"Path to get to upon connecting to the server")

var jsonFlag = flag.Bool("json", true,
	"Print the output in JSON instead of protobuf")

func main() {
	username, password, subscriptions, addrs, opts := client.ParseFlags()

	if *getFlag != "" {
		c := client.New(username, password, addrs[0], opts)
		for _, notification := range c.Get(*getFlag) {
			var notifStr string
			if *jsonFlag {
				var err error
				if notifStr, err = openconfig.NotificationToJSON(notification); err != nil {
					glog.Fatal(err)
				}
			} else {
				notifStr = notification.String()
			}
			fmt.Println(notifStr)

		}
		return
	}

	publish := func(addr string, message proto.Message) {
		resp, ok := message.(*pb.SubscribeResponse)
		if !ok {
			glog.Errorf("Unexpected type of message: %T", message)
			return
		}
		if resp.GetHeartbeat() != nil && !glog.V(1) {
			return // Log heartbeats with verbose logging only.
		}
		var respTxt string
		var err error
		if *jsonFlag {
			respTxt, err = openconfig.SubscribeResponseToJSON(resp)
			if err != nil {
				glog.Fatal(err)
			}
		} else {
			respTxt = proto.MarshalTextString(resp)
		}
		fmt.Println(respTxt)
	}

	wg := new(sync.WaitGroup)
	for _, addr := range addrs {
		wg.Add(1)
		c := client.New(username, password, addr, opts)
		go c.Subscribe(wg, subscriptions, publish)
	}
	wg.Wait()
}
