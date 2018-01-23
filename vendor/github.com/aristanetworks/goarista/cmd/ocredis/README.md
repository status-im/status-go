# ocredis

This is a client for the OpenConfig gRPC interface that publishes data to
Redis.  Values are stored in JSON.  Every update is pushed to Redis twice:

1. as a [hash map](http://redis.io/topics/data-types-intro#hashes) update,
   where the path in Redis is the path to the entity or collection (aka
   container or list, in YANG speak) and the keys of the hash are the
   attributes (leaf names, in YANG speak).
2. as a [`PUBLISH`](http://redis.io/commands/publish) command sent onto
   the path to the entity or collection, so that consumers can receive
   updates in a streaming fashion from Redis.

## Usage

See the `-help` output, but here's an example to push all the temperature
sensors into Redis.  You can also not pass any `-subscribe` flag to push
_everything_ into Redis.
```
ocredis -subscribe /Sysdb/environment/temperature -addrs <switch-hostname>:6042 -redis <redis-hostname>:6379
```
