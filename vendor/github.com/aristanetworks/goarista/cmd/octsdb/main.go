// Copyright (C) 2016  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

// The octsdb tool pushes OpenConfig telemetry to OpenTSDB.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"strconv"
	"strings"
	"sync"

	"github.com/aristanetworks/glog"
	"github.com/aristanetworks/goarista/openconfig/client"
	"github.com/golang/protobuf/proto"
	"github.com/openconfig/reference/rpc/openconfig"
)

func main() {
	tsdbFlag := flag.String("tsdb", "",
		"Address of the OpenTSDB server where to push telemetry to")
	textFlag := flag.Bool("text", false,
		"Print the output as simple text")
	configFlag := flag.String("config", "",
		"Config to turn OpenConfig telemetry into OpenTSDB put requests")
	username, password, subscriptions, addrs, opts := client.ParseFlags()

	if !(*tsdbFlag != "" || *textFlag) {
		glog.Fatal("Specify the address of the OpenTSDB server to write to with -tsdb")
	} else if *configFlag == "" {
		glog.Fatal("Specify a JSON configuration file with -config")
	}

	config, err := loadConfig(*configFlag)
	if err != nil {
		glog.Fatal(err)
	}
	// Ignore the default "subscribe-to-everything" subscription of the
	// -subscribe flag.
	if subscriptions[0] == "" {
		subscriptions = subscriptions[1:]
	}
	// Add the subscriptions from the config file.
	subscriptions = append(subscriptions, config.Subscriptions...)

	var c OpenTSDBConn
	if *textFlag {
		c = newTextDumper()
	} else {
		// TODO: support HTTP(S).
		c = newTelnetClient(*tsdbFlag)
	}

	wg := new(sync.WaitGroup)
	for _, addr := range addrs {
		wg.Add(1)
		publish := func(addr string, message proto.Message) {
			resp, ok := message.(*openconfig.SubscribeResponse)
			if !ok {
				glog.Errorf("Unexpected type of message: %T", message)
				return
			}
			if notif := resp.GetUpdate(); notif != nil {
				pushToOpenTSDB(addr, c, config, notif)
			}
		}
		c := client.New(username, password, addr, opts)
		go c.Subscribe(wg, subscriptions, publish)
	}
	wg.Wait()
}

func pushToOpenTSDB(addr string, conn OpenTSDBConn, config *Config,
	notif *openconfig.Notification) {

	if notif.Timestamp <= 0 {
		glog.Fatalf("Invalid timestamp %d in %s", notif.Timestamp, notif)
	}

	host := addr[:strings.IndexRune(addr, ':')]
	prefix := "/" + strings.Join(notif.Prefix.Element, "/")
	for _, update := range notif.Update {
		if update.Value == nil || update.Value.Type != openconfig.Type_JSON {
			glog.V(9).Infof("Ignoring incompatible update value in %s", update)
			continue
		}

		value := parseValue(update)
		if value == nil {
			glog.V(9).Infof("Ignoring non-numeric value in %s", update)
			continue
		}

		path := prefix + "/" + strings.Join(update.Path.Element, "/")
		metricName, tags := config.Match(path)
		if metricName == "" {
			glog.V(8).Infof("Ignoring unmatched update at %s: %+v", path, update.Value)
			continue
		}
		tags["host"] = host

		conn.Put(&DataPoint{
			Metric:    metricName,
			Timestamp: uint64(notif.Timestamp),
			Value:     value,
			Tags:      tags,
		})
	}
}

// parseValue returns the integer or floating point value of the given update,
// or nil if it's not a numerical update.
func parseValue(update *openconfig.Update) (value interface{}) {
	decoder := json.NewDecoder(bytes.NewReader(update.Value.Value))
	decoder.UseNumber()
	err := decoder.Decode(&value)
	if err != nil {
		glog.Fatalf("Malformed JSON update %q in %s", update.Value.Value, update)
	}
	num, ok := value.(json.Number)
	if !ok {
		return nil
	}
	// Convert our json.Number to either an int64, uint64, or float64.
	if value, err = num.Int64(); err != nil {
		// num is either a large unsigned integer or a floating point.
		if strings.Contains(err.Error(), "value out of range") { // Sigh.
			value, err = strconv.ParseUint(num.String(), 10, 64)
		} else {
			value, err = num.Float64()
			if err != nil {
				glog.Fatalf("Malformed JSON number %q in %s", num, update)
			}
		}
	}
	return
}
