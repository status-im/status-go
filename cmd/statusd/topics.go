package main

import (
	"errors"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/status-im/status-go/geth/params"
)

type topicsFlag []discv5.Topic

func (f *topicsFlag) String() string {
	return "discv5 topics"
}

func (f *topicsFlag) Set(value string) error {
	*f = append(*f, discv5.Topic(strings.TrimSpace(value)))
	return nil
}

type topicLimitsFlag map[discv5.Topic]params.Limits

func (f *topicLimitsFlag) String() string {
	return "disv5 topics to limits map"
}

func (f *topicLimitsFlag) Set(value string) error {
	parts := strings.Split(strings.TrimSpace(value), "=")
	if len(parts) != 2 {
		return errors.New("topic must be separated by '=' from limits, e.g. 'topic1=1,1'")
	}
	limits := strings.Split(parts[1], ",")
	if len(limits) != 2 {
		return errors.New("min and max limit must be set, e.g. 'topic1=1,1'")
	}
	minL, err := strconv.Atoi(limits[0])
	if err != nil {
		return err
	}
	maxL, err := strconv.Atoi(limits[1])
	if err != nil {
		return err
	}
	(*f)[discv5.Topic(parts[0])] = params.Limits{minL, maxL}
	return nil
}
