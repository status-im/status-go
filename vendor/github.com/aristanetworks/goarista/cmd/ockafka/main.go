// Copyright (C) 2016  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

// The occlient tool is a client for the gRPC service for getting and setting the
// OpenConfig configuration and state of a network device.
package main

import (
	"flag"
	"fmt"
	"strings"
	"sync"

	"github.com/Shopify/sarama"
	"github.com/aristanetworks/glog"
	"github.com/aristanetworks/goarista/kafka"
	"github.com/aristanetworks/goarista/kafka/openconfig"
	"github.com/aristanetworks/goarista/kafka/producer"
	"github.com/aristanetworks/goarista/openconfig/client"
	"github.com/golang/protobuf/proto"
)

var keysFlag = flag.String("kafkakeys", "",
	"Keys for kafka messages (comma-separated, default: the value of -addrs")

func newProducer(addresses []string, topic, key, dataset string) (producer.Producer,
	error) {

	glog.Infof("Connected to Kafka brokers at %s", addresses)
	encodedKey := sarama.StringEncoder(key)
	p, err := producer.New(topic, nil, encodedKey, dataset,
		openconfig.ElasticsearchMessageEncoder, addresses, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to create Kafka producer: %s", err)
	}
	return p, nil
}

func main() {
	username, password, subscriptions, grpcAddrs, opts := client.ParseFlags()

	if *keysFlag == "" {
		*keysFlag = strings.Join(grpcAddrs, ",")
	}
	keys := strings.Split(*keysFlag, ",")
	if len(grpcAddrs) != len(keys) {
		glog.Fatal("Please provide the same number of addresses and Kafka keys")
	}
	addresses := strings.Split(*kafka.Addresses, ",")
	wg := new(sync.WaitGroup)
	for i, grpcAddr := range grpcAddrs {
		key := keys[i]
		p, err := newProducer(addresses, *kafka.Topic, key, grpcAddr)
		if err != nil {
			glog.Fatal(err)
		} else {
			glog.Infof("Initialized Kafka producer for %s", grpcAddr)
		}
		publish := func(addr string, message proto.Message) {
			p.Write(message)
		}
		wg.Add(1)
		go p.Run()
		c := client.New(username, password, grpcAddr, opts)
		go c.Subscribe(wg, subscriptions, publish)
	}
	wg.Wait()
}
