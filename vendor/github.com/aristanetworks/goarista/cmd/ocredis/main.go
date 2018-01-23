// Copyright (C) 2016  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

// The ocredis tool is a client for the OpenConfig gRPC interface that
// subscribes to state and pushes it to Redis, using Redis' support for hash
// maps and for publishing events that can be subscribed to.
package main

import (
	"encoding/json"
	"flag"
	"strings"
	"sync"

	"github.com/aristanetworks/glog"
	occlient "github.com/aristanetworks/goarista/openconfig/client"
	"github.com/golang/protobuf/proto"
	"github.com/openconfig/reference/rpc/openconfig"
	redis "gopkg.in/redis.v4"
)

var clusterMode = flag.Bool("cluster", false, "Whether the redis server is a cluster")

var redisFlag = flag.String("redis", "",
	"Comma separated list of Redis servers to push updates to")

var redisPassword = flag.String("redispass", "", "Password of redis server/cluster")

// baseClient allows us to represent both a redis.Client and redis.ClusterClient.
type baseClient interface {
	Close() error
	ClusterInfo() *redis.StringCmd
	HDel(string, ...string) *redis.IntCmd
	HMSet(string, map[string]string) *redis.StatusCmd
	Ping() *redis.StatusCmd
	Pipelined(func(*redis.Pipeline) error) ([]redis.Cmder, error)
	Publish(string, string) *redis.IntCmd
}

var client baseClient

func main() {
	username, password, subscriptions, hostAddrs, opts := occlient.ParseFlags()
	if *redisFlag == "" {
		glog.Fatal("Specify the address of the Redis server to write to with -redis")
	}

	redisAddrs := strings.Split(*redisFlag, ",")
	if !*clusterMode && len(redisAddrs) > 1 {
		glog.Fatal("Please pass only 1 redis address in noncluster mode or enable cluster mode")
	}

	if *clusterMode {
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    redisAddrs,
			Password: *redisPassword,
		})
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:     *redisFlag,
			Password: *redisPassword,
		})
	}
	defer client.Close()

	// TODO: Figure out ways to handle being in the wrong mode:
	// Connecting to cluster in non cluster mode - we get a MOVED error on the first HMSET
	// Connecting to a noncluster in cluster mode - we get stuck forever
	_, err := client.Ping().Result()
	if err != nil {
		glog.Fatal("Failed to connect to client: ", err)
	}

	ocPublish := func(addr string, message proto.Message) {
		resp, ok := message.(*openconfig.SubscribeResponse)
		if !ok {
			glog.Errorf("Unexpected type of message: %T", message)
			return
		}
		if notif := resp.GetUpdate(); notif != nil {
			bufferToRedis(addr, notif)
		}
	}

	wg := new(sync.WaitGroup)
	for _, hostAddr := range hostAddrs {
		wg.Add(1)
		c := occlient.New(username, password, hostAddr, opts)
		go c.Subscribe(wg, subscriptions, ocPublish)
	}
	wg.Wait()
}

type redisData struct {
	key   string
	hmset map[string]string
	hdel  []string
	pub   map[string]interface{}
}

func bufferToRedis(addr string, notif *openconfig.Notification) {
	path := addr + "/" + joinPath(notif.Prefix)
	data := &redisData{key: path}

	if len(notif.Update) != 0 {
		hmset := make(map[string]string, len(notif.Update))

		// Updates to publish on the pub/sub.
		pub := make(map[string]interface{}, len(notif.Update))
		for _, update := range notif.Update {
			key := joinPath(update.Path)
			value := convertUpdate(update)
			pub[key] = value
			marshaledValue, err := json.Marshal(value)
			if err != nil {
				glog.Fatalf("Failed to JSON marshal update %#v", update)
			}
			hmset[key] = string(marshaledValue)
		}
		data.hmset = hmset
		data.pub = pub
	}

	if len(notif.Delete) != 0 {
		hdel := make([]string, len(notif.Delete))
		for i, del := range notif.Delete {
			hdel[i] = joinPath(del)
		}
		data.hdel = hdel
	}
	pushToRedis(data)
}

func pushToRedis(data *redisData) {
	_, err := client.Pipelined(func(pipe *redis.Pipeline) error {
		if data.hmset != nil {
			if reply := client.HMSet(data.key, data.hmset); reply.Err() != nil {
				glog.Fatal("Redis HMSET error: ", reply.Err())
			}
			redisPublish(data.key, "updates", data.pub)
		}
		if data.hdel != nil {
			if reply := client.HDel(data.key, data.hdel...); reply.Err() != nil {
				glog.Fatal("Redis HDEL error: ", reply.Err())
			}
			redisPublish(data.key, "deletes", data.hdel)
		}
		return nil
	})
	if err != nil {
		glog.Fatal("Failed to send Pipelined commands: ", err)
	}
}

func redisPublish(path, kind string, payload interface{}) {
	js, err := json.Marshal(map[string]interface{}{
		"kind":    kind,
		"payload": payload,
	})
	if err != nil {
		glog.Fatalf("JSON error: %s", err)
	}
	if reply := client.Publish(path, string(js)); reply.Err() != nil {
		glog.Fatal("Redis PUBLISH error: ", reply.Err())
	}
}

func joinPath(path *openconfig.Path) string {
	return strings.Join(path.Element, "/")
}

func convertUpdate(update *openconfig.Update) interface{} {
	switch update.Value.Type {
	case openconfig.Type_JSON:
		var value interface{}
		err := json.Unmarshal(update.Value.Value, &value)
		if err != nil {
			glog.Fatalf("Malformed JSON update %q in %s", update.Value.Value, update)
		}
		return value
	case openconfig.Type_BYTES:
		return update.Value.Value
	default:
		glog.Fatalf("Unhandled type of value %v in %s", update.Value.Type, update)
		return nil
	}
}
