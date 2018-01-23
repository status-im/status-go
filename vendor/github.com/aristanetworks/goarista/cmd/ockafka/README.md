# ockafka

Client for the gRPC OpenConfig service for subscribing to the configuration and
state of a network device and feeding the stream to Kafka.

## Sample usage

Subscribe to all updates on the Arista device at `10.0.1.2` and stream to a local
Kafka instance:

```
ockafka -addrs 10.0.1.2
```

Subscribe to temperature sensors from 2 switches and stream to a remote Kafka instance:

```
ockafka -addrs 10.0.1.2,10.0.1.3 -kafkaaddrs kafka:9092 -subscribe /Sysdb/environment/temperature/status/tempSensor
```

Start in a container:
```
docker run aristanetworks/ockafka -addrs 10.0.1.1 -kafkaaddrs kafka:9092
```

## Kafka/Elastic integration demo
The following video demoes integration with Kafka and Elastic using [this Logstash instance](https://github.com/aristanetworks/docker-logstash):

[![video preview](http://img.youtube.com/vi/WsyFmxMwXYQ/0.jpg)](https://youtu.be/WsyFmxMwXYQ)
