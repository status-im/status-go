package utils

import "github.com/status-im/status-go/common"

func BridgeChannels[In any, Out any](in <-chan In, convert func(In) Out) <-chan Out {
	out := make(chan Out)
	go func() {
		defer common.LogOnPanic()
		defer close(out)
		for v := range in {
			out <- convert(v)
		}
	}()
	return out
}

func BridgeChannelsSlice[In any, Out any](in <-chan []In, convert func(In) Out) <-chan []Out {
	out := make(chan []Out)
	go func() {
		defer common.LogOnPanic()
		defer close(out)
		for v := range in {
			var converted []Out
			for _, item := range v {
				converted = append(converted, convert(item))
			}
			out <- converted
		}
	}()
	return out
}
