// Copyright (C) 2016  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package kafka

import (
	"os"
	"time"

	"github.com/Shopify/sarama"
	"github.com/aristanetworks/glog"
)

const (
	outOfBrokersBackoff = 30 * time.Second
	outOfBrokersRetries = 5
)

// NewClient returns a Kafka client
func NewClient(addresses []string) (sarama.Client, error) {
	config := sarama.NewConfig()
	hostname, err := os.Hostname()
	if err != nil {
		hostname = ""
	}
	config.ClientID = hostname
	config.Producer.Compression = sarama.CompressionSnappy
	config.Producer.Return.Successes = true

	var client sarama.Client
	retries := outOfBrokersRetries + 1
	for retries > 0 {
		client, err = sarama.NewClient(addresses, config)
		retries--
		if err == sarama.ErrOutOfBrokers {
			glog.Errorf("Can't connect to the Kafka cluster at %s (%d retries left): %s",
				addresses, retries, err)
			time.Sleep(outOfBrokersBackoff)
		} else {
			break
		}
	}
	return client, err
}
