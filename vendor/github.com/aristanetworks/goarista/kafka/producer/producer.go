// Copyright (C) 2016  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package producer

import (
	"expvar"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Shopify/sarama"
	"github.com/aristanetworks/glog"
	"github.com/aristanetworks/goarista/kafka"
	"github.com/aristanetworks/goarista/kafka/openconfig"
	"github.com/aristanetworks/goarista/monitor"
	"github.com/golang/protobuf/proto"
)

// counter counts the number Sysdb clients we have, and is used to guarantee that we
// always have a unique name exported to expvar
var counter uint32

// MessageEncoder defines the encoding from topic, key, proto.Message to sarama.ProducerMessage
type MessageEncoder func(string, sarama.Encoder, string, proto.Message) (*sarama.ProducerMessage,
	error)

// Producer forwards messages recvd on a channel to kafka.
type Producer interface {
	Run()
	Write(proto.Message)
	Stop()
}

type producer struct {
	notifsChan    chan proto.Message
	kafkaProducer sarama.AsyncProducer
	topic         string
	key           sarama.Encoder
	dataset       string
	encoder       MessageEncoder
	done          chan struct{}
	wg            sync.WaitGroup

	// Used for monitoring
	histogram    *monitor.Histogram
	numSuccesses monitor.Uint
	numFailures  monitor.Uint
}

// New creates new Kafka producer
func New(topic string, notifsChan chan proto.Message,
	key sarama.Encoder, dataset string, encoder MessageEncoder,
	kafkaAddresses []string, kafkaConfig *sarama.Config) (Producer, error) {
	if notifsChan == nil {
		notifsChan = make(chan proto.Message)
	}

	if kafkaConfig == nil {
		kafkaConfig := sarama.NewConfig()
		hostname, err := os.Hostname()
		if err != nil {
			hostname = ""
		}
		kafkaConfig.ClientID = hostname
		kafkaConfig.Producer.Compression = sarama.CompressionSnappy
		kafkaConfig.Producer.Return.Successes = true
	}

	kafkaProducer, err := sarama.NewAsyncProducer(kafkaAddresses, kafkaConfig)
	if err != nil {
		return nil, err
	}

	// Setup monitoring structures
	histName := "kafkaProducerHistogram"
	statsName := "messagesStats"
	if id := atomic.AddUint32(&counter, 1); id > 1 {
		histName = fmt.Sprintf("%s-%d", histName, id)
		statsName = fmt.Sprintf("%s-%d", statsName, id)
	}
	hist := monitor.NewHistogram(histName, 32, 0.3, 1000, 0)
	statsMap := expvar.NewMap(statsName)

	p := &producer{
		notifsChan:    notifsChan,
		kafkaProducer: kafkaProducer,
		topic:         topic,
		key:           key,
		dataset:       dataset,
		encoder:       encoder,
		done:          make(chan struct{}),
		wg:            sync.WaitGroup{},
		histogram:     hist,
	}

	statsMap.Set("successes", &p.numSuccesses)
	statsMap.Set("failures", &p.numFailures)

	return p, nil
}

func (p *producer) Run() {
	p.wg.Add(2)
	go p.handleSuccesses()
	go p.handleErrors()

	p.wg.Add(1)
	defer p.wg.Done()
	for {
		select {
		case batch, open := <-p.notifsChan:
			if !open {
				return
			}
			err := p.produceNotification(batch)
			if err != nil {
				if _, ok := err.(openconfig.UnhandledSubscribeResponseError); !ok {
					panic(err)
				}
			}
		case <-p.done:
			return
		}
	}
}

func (p *producer) Write(m proto.Message) {
	p.notifsChan <- m
}

func (p *producer) Stop() {
	close(p.done)
	p.kafkaProducer.Close()
	p.wg.Wait()
}

func (p *producer) produceNotification(protoMessage proto.Message) error {
	message, err := p.encoder(p.topic, p.key, p.dataset, protoMessage)
	if err != nil {
		return err
	}
	select {
	case p.kafkaProducer.Input() <- message:
		glog.V(9).Infof("Message produced to Kafka: %s", message)
		return nil
	case <-p.done:
		return nil
	}
}

// handleSuccesses reads from the producer's successes channel and collects some
// information for monitoring
func (p *producer) handleSuccesses() {
	defer p.wg.Done()
	for msg := range p.kafkaProducer.Successes() {
		metadata := msg.Metadata.(kafka.Metadata)
		// TODO: Add a monotonic clock source when one becomes available
		p.histogram.UpdateLatencyValues(metadata.StartTime, time.Now())
		p.numSuccesses.Add(uint64(metadata.NumMessages))
	}
}

// handleErrors reads from the producer's errors channel and collects some information
// for monitoring
func (p *producer) handleErrors() {
	defer p.wg.Done()
	for msg := range p.kafkaProducer.Errors() {
		metadata := msg.Msg.Metadata.(kafka.Metadata)
		// TODO: Add a monotonic clock source when one becomes available
		p.histogram.UpdateLatencyValues(metadata.StartTime, time.Now())
		glog.Errorf("Kafka Producer error: %s", msg.Error())
		p.numFailures.Add(uint64(metadata.NumMessages))
	}
}
