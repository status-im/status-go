package rpclimiter

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/logutils"
)

const (
	defaultMaxRequestsPerSecond = 50
	minRequestsPerSecond        = 20
	requestsPerSecondStep       = 10

	tickerInterval  = 1 * time.Second
	LimitInfinitely = 0
)

var (
	ErrRequestsOverLimit = errors.New("number of requests over limit")
)

type callerOnWait struct {
	requests int
	ch       chan bool
}

type LimitsStorage interface {
	Get(tag string) (*LimitData, error)
	Set(data *LimitData) error
	Delete(tag string) error
}

type InMemRequestsMapStorage struct {
	data sync.Map
}

func NewInMemRequestsMapStorage() *InMemRequestsMapStorage {
	return &InMemRequestsMapStorage{}
}

func (s *InMemRequestsMapStorage) Get(tag string) (*LimitData, error) {
	data, ok := s.data.Load(tag)
	if !ok {
		return nil, nil
	}

	return data.(*LimitData), nil
}

func (s *InMemRequestsMapStorage) Set(data *LimitData) error {
	if data == nil {
		return fmt.Errorf("data is nil")
	}

	s.data.Store(data.Tag, data)
	return nil
}

func (s *InMemRequestsMapStorage) Delete(tag string) error {
	s.data.Delete(tag)
	return nil
}

type LimitsDBStorage struct {
	db *RPCLimiterDB
}

func NewLimitsDBStorage(db *sql.DB) *LimitsDBStorage {
	return &LimitsDBStorage{
		db: NewRPCLimiterDB(db),
	}
}

func (s *LimitsDBStorage) Get(tag string) (*LimitData, error) {
	return s.db.GetRPCLimit(tag)
}

func (s *LimitsDBStorage) Set(data *LimitData) error {
	if data == nil {
		return fmt.Errorf("data is nil")
	}

	limit, err := s.db.GetRPCLimit(data.Tag)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if limit == nil {
		return s.db.CreateRPCLimit(*data)
	}

	return s.db.UpdateRPCLimit(*data)
}

func (s *LimitsDBStorage) Delete(tag string) error {
	return s.db.DeleteRPCLimit(tag)
}

type LimitData struct {
	Tag       string
	CreatedAt time.Time
	Period    time.Duration
	MaxReqs   int
	NumReqs   int
}

type RequestLimiter interface {
	SetLimit(tag string, maxRequests int, interval time.Duration) error
	GetLimit(tag string) (*LimitData, error)
	DeleteLimit(tag string) error
	Allow(tag string) (bool, error)
}

type RPCRequestLimiter struct {
	storage LimitsStorage
	mu      sync.Mutex
}

func NewRequestLimiter(storage LimitsStorage) *RPCRequestLimiter {
	return &RPCRequestLimiter{
		storage: storage,
	}
}

func (rl *RPCRequestLimiter) SetLimit(tag string, maxRequests int, interval time.Duration) error {
	err := rl.saveToStorage(tag, maxRequests, interval, 0, time.Now())
	if err != nil {
		logutils.ZapLogger().Error("Failed to save request data to storage", zap.Error(err))
		return err
	}

	return nil
}

func (rl *RPCRequestLimiter) GetLimit(tag string) (*LimitData, error) {
	data, err := rl.storage.Get(tag)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (rl *RPCRequestLimiter) DeleteLimit(tag string) error {
	err := rl.storage.Delete(tag)
	if err != nil {
		logutils.ZapLogger().Error("Failed to delete request data from storage", zap.Error(err))
		return err
	}

	return nil
}

func (rl *RPCRequestLimiter) saveToStorage(tag string, maxRequests int, interval time.Duration, numReqs int, timestamp time.Time) error {
	data := &LimitData{
		Tag:       tag,
		CreatedAt: timestamp,
		Period:    interval,
		MaxReqs:   maxRequests,
		NumReqs:   numReqs,
	}

	err := rl.storage.Set(data)
	if err != nil {
		logutils.ZapLogger().Error("Failed to save request data to storage", zap.Error(err))
		return err
	}

	return nil
}

func (rl *RPCRequestLimiter) Allow(tag string) (bool, error) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	data, err := rl.storage.Get(tag)
	if err != nil {
		return true, err
	}

	if data == nil {
		return true, nil
	}

	// Check if the interval has passed and reset the number of requests
	if time.Since(data.CreatedAt) >= data.Period && data.Period.Milliseconds() != LimitInfinitely {
		err = rl.saveToStorage(tag, data.MaxReqs, data.Period, 0, time.Now())
		if err != nil {
			return true, err
		}

		return true, nil
	}

	// Check if a number of requests is over the limit within the interval
	if time.Since(data.CreatedAt) < data.Period || data.Period.Milliseconds() == LimitInfinitely {
		if data.NumReqs >= data.MaxReqs {
			logutils.ZapLogger().Info("Number of requests over limit",
				zap.String("tag", tag),
				zap.Int("numReqs", data.NumReqs),
				zap.Int("maxReqs", data.MaxReqs),
				zap.Duration("period", data.Period),
				zap.Time("createdAt", data.CreatedAt.UTC()))
			return false, ErrRequestsOverLimit
		}

		return true, rl.saveToStorage(tag, data.MaxReqs, data.Period, data.NumReqs+1, data.CreatedAt)
	}

	// Reset the number of requests if the interval has passed
	return true, rl.saveToStorage(tag, data.MaxReqs, data.Period, 0, time.Now()) // still allow the request if failed to save as not critical
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
		defer gocommon.LogOnPanic()
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
