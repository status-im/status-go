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

// InMemRequestsStorage is an in-memory dummy implementation of RequestsStorage
type InMemRequestsStorage struct {
	data RequestData
}

func NewInMemRequestsStorage() *InMemRequestsStorage {
	return &InMemRequestsStorage{}
}

func (s *InMemRequestsStorage) Get(tag string) (RequestData, error) {
	return s.data, nil
}

func (s *InMemRequestsStorage) Set(data RequestData) error {
	s.data = data
	return nil
}

type RequestData struct {
	Tag       string
	CreatedAt time.Time
	Period    time.Duration
	MaxReqs   int
	NumReqs   int
}

type RequestLimiter interface {
	SetMaxRequests(tag string, maxRequests int, interval time.Duration) error
	GetMaxRequests(tag string) (RequestData, error)
	IsLimitReached(tag string) (bool, error)
}

type RPCRequestLimiter struct {
	storage RequestsStorage
}

func NewRequestLimiter(storage RequestsStorage) *RPCRequestLimiter {
	return &RPCRequestLimiter{
		storage: storage,
	}
}

func (rl *RPCRequestLimiter) SetMaxRequests(tag string, maxRequests int, interval time.Duration) error {
	err := rl.saveToStorage(tag, maxRequests, interval, 0, time.Now())
	if err != nil {
		log.Error("Failed to save request data to storage", "error", err)
		return err
	}

	return nil
}

func (rl *RPCRequestLimiter) GetMaxRequests(tag string) (RequestData, error) {
	data, err := rl.storage.Get(tag)
	if err != nil {
		log.Error("Failed to get request data from storage", "error", err, "tag", tag)
		return RequestData{}, err
	}

	return data, nil
}

func (rl *RPCRequestLimiter) saveToStorage(tag string, maxRequests int, interval time.Duration, numReqs int, timestamp time.Time) error {
	data := RequestData{
		Tag:       tag,
		CreatedAt: timestamp,
		Period:    interval,
		MaxReqs:   maxRequests,
		NumReqs:   numReqs,
	}

	err := rl.storage.Set(data)
	if err != nil {
		log.Error("Failed to save request data to storage", "error", err)
		return err
	}

	return nil
}

func (rl *RPCRequestLimiter) IsLimitReached(tag string) (bool, error) {
	data, err := rl.storage.Get(tag)
	if err != nil {
		return false, err
	}

	// Check if a number of requests is over the limit within the interval
	if time.Since(data.CreatedAt) < data.Period {
		if data.NumReqs >= data.MaxReqs {
			return true, nil
		}

		err := rl.saveToStorage(tag, data.MaxReqs, data.Period, data.NumReqs+1, data.CreatedAt)
		if err != nil {
			return false, err
		}

		return false, nil
	}

	// Reset the number of requests if the interval has passed
	err = rl.saveToStorage(tag, data.MaxReqs, data.Period, 0, time.Now())
	if err != nil {
		return false, err
	}

	return false, nil
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
