package sign

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/geth/account"
	"github.com/stretchr/testify/suite"
)

const (
	correctPassword = "password-correct"
	wrongPassword   = "password-wrong"
)

func testVerifyFunc(password string) (*account.SelectedExtKey, error) {
	if password == correctPassword {
		return nil, nil
	}

	return nil, keystore.ErrDecrypt
}

func TestPendingRequestsSuite(t *testing.T) {
	suite.Run(t, new(PendingRequestsSuite))
}

type PendingRequestsSuite struct {
	suite.Suite
	pendingRequests *PendingRequests
}

func (s *PendingRequestsSuite) SetupTest() {
	s.pendingRequests = NewPendingRequests()
}

func (s *PendingRequestsSuite) defaultCompleteFunc() CompleteFunc {
	hash := gethcommon.Hash{1}
	return func(acc *account.SelectedExtKey, password string) (Response, error) {
		s.Nil(acc, "account should be `nil`")
		s.Equal(correctPassword, password)
		return hash.Bytes(), nil
	}
}

func (s *PendingRequestsSuite) delayedCompleteFunc() CompleteFunc {
	hash := gethcommon.Hash{1}
	return func(acc *account.SelectedExtKey, password string) (Response, error) {
		time.Sleep(10 * time.Millisecond)
		s.Nil(acc, "account should be `nil`")
		s.Equal(correctPassword, password)
		return hash.Bytes(), nil
	}
}

func (s *PendingRequestsSuite) errorCompleteFunc(err error) CompleteFunc {
	hash := gethcommon.Hash{1}
	return func(acc *account.SelectedExtKey, password string) (Response, error) {
		s.Nil(acc, "account should be `nil`")
		return hash.Bytes(), err
	}
}

func (s *PendingRequestsSuite) TestGet() {
	req, err := s.pendingRequests.Add(context.Background(), "", nil, s.defaultCompleteFunc())
	s.NoError(err)
	for i := 2; i > 0; i-- {
		actualRequest, err := s.pendingRequests.Get(req.ID)
		s.NoError(err)
		s.Equal(req, actualRequest)
	}
}

func (s *PendingRequestsSuite) testComplete(password string, hash gethcommon.Hash, completeFunc CompleteFunc) (string, error) {
	req, err := s.pendingRequests.Add(context.Background(), "", nil, completeFunc)
	s.NoError(err)

	s.True(s.pendingRequests.Has(req.ID), "sign request should exist")

	result := s.pendingRequests.Approve(req.ID, password, testVerifyFunc)

	if s.pendingRequests.Has(req.ID) {
		// transient error
		s.Equal(EmptyResponse, result.Response, "no hash should be sent")
	} else {
		s.Equal(hash.Bytes(), result.Response.Bytes(), "hashes should match")
	}

	return req.ID, result.Error
}

func (s *PendingRequestsSuite) TestCompleteSuccess() {
	id, err := s.testComplete(correctPassword, gethcommon.Hash{1}, s.defaultCompleteFunc())
	s.NoError(err, "no errors should be there")

	s.False(s.pendingRequests.Has(id), "sign request should not exist")
}

func (s *PendingRequestsSuite) TestCompleteTransientError() {
	hash := gethcommon.Hash{}
	id, err := s.testComplete(wrongPassword, hash, s.errorCompleteFunc(keystore.ErrDecrypt))
	s.Equal(keystore.ErrDecrypt, err, "error value should be preserved")

	s.True(s.pendingRequests.Has(id))
	// verify that you are able to re-approve it after a transient error
	_, err = s.pendingRequests.tryLock(id)
	s.NoError(err)
}

func (s *PendingRequestsSuite) TestCompleteError() {
	hash := gethcommon.Hash{1}
	expectedError := errors.New("test")

	id, err := s.testComplete(correctPassword, hash, s.errorCompleteFunc(expectedError))

	s.Equal(expectedError, err, "error value should be preserved")

	s.False(s.pendingRequests.Has(id))
}

func (s PendingRequestsSuite) TestMultipleComplete() {
	id, err := s.testComplete(correctPassword, gethcommon.Hash{1}, s.defaultCompleteFunc())
	s.NoError(err, "no errors should be there")

	result := s.pendingRequests.Approve(id, correctPassword, testVerifyFunc)

	s.Equal(ErrSignReqNotFound, result.Error)
}

func (s PendingRequestsSuite) TestConcurrentComplete() {
	req, err := s.pendingRequests.Add(context.Background(), "", nil, s.delayedCompleteFunc())
	s.NoError(err)

	s.True(s.pendingRequests.Has(req.ID), "sign request should exist")

	approved := 0
	tried := 0

	for i := 10; i > 0; i-- {
		go func() {
			result := s.pendingRequests.Approve(req.ID, correctPassword, testVerifyFunc)
			if result.Error == nil {
				approved++
			}
			tried++
		}()
	}

	s.pendingRequests.Wait(req.ID, 10*time.Second)

	s.False(s.pendingRequests.Has(req.ID), "sign request should exist")

	s.Equal(approved, 1, "request should be approved only once")
	s.Equal(tried, 10, "request should be tried to approve 10 times")
}

func (s PendingRequestsSuite) TestWaitSuccess() {
	req, err := s.pendingRequests.Add(context.Background(), "", nil, s.defaultCompleteFunc())
	s.NoError(err)

	s.True(s.pendingRequests.Has(req.ID), "sign request should exist")

	go func() {
		result := s.pendingRequests.Approve(req.ID, correctPassword, testVerifyFunc)
		s.NoError(result.Error)
	}()

	result := s.pendingRequests.Wait(req.ID, 1*time.Second)
	s.NoError(result.Error)
}

func (s PendingRequestsSuite) TestDiscard() {
	req, err := s.pendingRequests.Add(context.Background(), "", nil, s.defaultCompleteFunc())
	s.NoError(err)

	s.True(s.pendingRequests.Has(req.ID), "sign request should exist")

	s.Equal(ErrSignReqNotFound, s.pendingRequests.Discard(""))

	go func() {
		// enough to make it be called after Wait
		time.Sleep(time.Millisecond)
		s.NoError(s.pendingRequests.Discard(req.ID))
	}()

	result := s.pendingRequests.Wait(req.ID, 1*time.Second)
	s.Equal(ErrSignReqDiscarded, result.Error)
}

func (s PendingRequestsSuite) TestWaitFail() {
	expectedError := errors.New("test-wait-fail")
	req, err := s.pendingRequests.Add(context.Background(), "", nil, s.errorCompleteFunc(expectedError))
	s.NoError(err)

	s.True(s.pendingRequests.Has(req.ID), "sign request should exist")

	go func() {
		result := s.pendingRequests.Approve(req.ID, correctPassword, testVerifyFunc)
		s.Equal(expectedError, result.Error)
	}()

	result := s.pendingRequests.Wait(req.ID, 1*time.Second)
	s.Equal(expectedError, result.Error)
}

func (s PendingRequestsSuite) TestWaitTimeout() {
	req, err := s.pendingRequests.Add(context.Background(), "", nil, s.delayedCompleteFunc())
	s.NoError(err)

	s.True(s.pendingRequests.Has(req.ID), "sign request should exist")

	go func() {
		result := s.pendingRequests.Approve(req.ID, correctPassword, testVerifyFunc)
		s.NoError(result.Error)
	}()

	result := s.pendingRequests.Wait(req.ID, 0*time.Second)
	s.Equal(result.Error, ErrSignReqTimedOut)
}
