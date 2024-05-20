package chain

import (
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
)

const (
	defaultMaxRequestsPerSecond = 50
	minRequestsPerSecond        = 20
	requestsPerSecondStep       = 10

	tickerInterval = 1 * time.Second
)

var (
	ErrRequestsOverLimit = fmt.Errorf("number of requests over limit")
)

type callerOnWait struct {
	requests int
	ch       chan bool
}

type RequestsStorage interface {
	Get(tag string) (RequestData, error)
	Set(data RequestData) error
}

type RequestData struct {
	Tag       string
	CreatedAt time.Time
	Period    time.Duration
}

type RequestLimiter interface {
	SetMaxRequests(tag string, maxRequests int, interval time.Duration)
	IsLimitReached(tag string) bool
}

type RPCRequestLimiter struct {
	storage RequestsStorage
}

func NewRequestLimiter(storage RequestsStorage) *RPCRequestLimiter {
	return &RPCRequestLimiter{
		storage: storage,
	}
}

func (rl *RPCRequestLimiter) SetMaxRequests(tag string, maxRequests int, interval time.Duration) {
	err := rl.saveToStorage(tag, maxRequests, interval)
	if err != nil {
		log.Error("Failed to save request data to storage", "error", err)
		return
	}

	// Set max requests logic here
}

func (rl *RPCRequestLimiter) saveToStorage(tag string, maxRequests int, interval time.Duration) error {
	data := RequestData{
		Tag:       tag,
		CreatedAt: time.Now(),
		Period:    interval,
	}

	err := rl.storage.Set(data)
	if err != nil {
		return err
	}

	return nil
}

func (rl *RPCRequestLimiter) IsLimitReached(tag string) bool {
	data, err := rl.storage.Get(tag)
	if err != nil {
		log.Error("Failed to get request data from storage", "error", err, "tag", tag)
		return false
	}

	return time.Since(data.CreatedAt) >= data.Period
}

type RPCRpsLimiter struct {
	uuid uuid.UUID

	maxRequestsPerSecond      int
	maxRequestsPerSecondMutex sync.RWMutex

	requestsMadeWithinSecond      int
	requestsMadeWithinSecondMutex sync.RWMutex

	callersOnWaitForRequests      []callerOnWait
	callersOnWaitForRequestsMutex sync.RWMutex

	quit chan bool
}

func NewRPCRpsLimiter() *RPCRpsLimiter {

	limiter := RPCRpsLimiter{
		uuid:                 uuid.New(),
		maxRequestsPerSecond: defaultMaxRequestsPerSecond,
		quit:                 make(chan bool),
	}

	limiter.start()

	return &limiter
}

func (rl *RPCRpsLimiter) ReduceLimit() {
	rl.maxRequestsPerSecondMutex.Lock()
	defer rl.maxRequestsPerSecondMutex.Unlock()
	if rl.maxRequestsPerSecond <= minRequestsPerSecond {
		return
	}
	rl.maxRequestsPerSecond = rl.maxRequestsPerSecond - requestsPerSecondStep
}

func (rl *RPCRpsLimiter) start() {
	ticker := time.NewTicker(tickerInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				{
					rl.requestsMadeWithinSecondMutex.Lock()
					oldrequestsMadeWithinSecond := rl.requestsMadeWithinSecond
					if rl.requestsMadeWithinSecond != 0 {
						rl.requestsMadeWithinSecond = 0
					}
					rl.requestsMadeWithinSecondMutex.Unlock()
					if oldrequestsMadeWithinSecond == 0 {
						continue
					}
				}

				rl.callersOnWaitForRequestsMutex.Lock()
				numOfRequestsToMakeAvailable := rl.maxRequestsPerSecond
				for {
					if numOfRequestsToMakeAvailable == 0 || len(rl.callersOnWaitForRequests) == 0 {
						break
					}

					var index = -1
					for i := 0; i < len(rl.callersOnWaitForRequests); i++ {
						if rl.callersOnWaitForRequests[i].requests <= numOfRequestsToMakeAvailable {
							index = i
							break
						}
					}

					if index == -1 {
						break
					}

					callerOnWait := rl.callersOnWaitForRequests[index]
					numOfRequestsToMakeAvailable -= callerOnWait.requests
					rl.callersOnWaitForRequests = append(rl.callersOnWaitForRequests[:index], rl.callersOnWaitForRequests[index+1:]...)

					callerOnWait.ch <- true
				}
				rl.callersOnWaitForRequestsMutex.Unlock()

			case <-rl.quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func (rl *RPCRpsLimiter) Stop() {
	rl.quit <- true
	close(rl.quit)
	for _, callerOnWait := range rl.callersOnWaitForRequests {
		close(callerOnWait.ch)
	}
	rl.callersOnWaitForRequests = nil
}

func (rl *RPCRpsLimiter) WaitForRequestsAvailability(requests int) error {
	if requests > rl.maxRequestsPerSecond {
		return ErrRequestsOverLimit
	}

	{
		rl.requestsMadeWithinSecondMutex.Lock()
		if rl.requestsMadeWithinSecond+requests <= rl.maxRequestsPerSecond {
			rl.requestsMadeWithinSecond += requests
			rl.requestsMadeWithinSecondMutex.Unlock()
			return nil
		}
		rl.requestsMadeWithinSecondMutex.Unlock()
	}

	callerOnWait := callerOnWait{
		requests: requests,
		ch:       make(chan bool),
	}

	{
		rl.callersOnWaitForRequestsMutex.Lock()
		rl.callersOnWaitForRequests = append(rl.callersOnWaitForRequests, callerOnWait)
		rl.callersOnWaitForRequestsMutex.Unlock()
	}

	<-callerOnWait.ch

	close(callerOnWait.ch)

	rl.requestsMadeWithinSecondMutex.Lock()
	rl.requestsMadeWithinSecond += requests
	rl.requestsMadeWithinSecondMutex.Unlock()

	return nil
}
