package queue

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/geth/common"
	"github.com/stretchr/testify/suite"
)

func TestQueueTestSuite(t *testing.T) {
	suite.Run(t, new(QueueTestSuite))
}

type QueueTestSuite struct {
	suite.Suite
	queue *TxQueue
}

func (s *QueueTestSuite) SetupTest() {
	s.queue = New()
	s.queue.Start()
}

func (s *QueueTestSuite) TearDownTest() {
	s.queue.Stop()
}

func (s *QueueTestSuite) TestLockInprogressTransaction() {
	tx := common.CreateTransaction(context.Background(), common.SendTxArgs{})
	s.NoError(s.queue.Enqueue(tx))
	enquedTx, err := s.queue.LockInprogress(tx.ID)
	s.NoError(err)
	s.Equal(tx, enquedTx)

	// verify that tx was marked as being inprogress
	_, err = s.queue.LockInprogress(tx.ID)
	s.Equal(ErrQueuedTxInProgress, err)
}

func (s *QueueTestSuite) TestGetTransaction() {
	tx := common.CreateTransaction(context.Background(), common.SendTxArgs{})
	s.NoError(s.queue.Enqueue(tx))
	for i := 2; i > 0; i-- {
		enquedTx, err := s.queue.Get(tx.ID)
		s.NoError(err)
		s.Equal(tx, enquedTx)
	}
}

func (s *QueueTestSuite) TestEnqueueProcessedTransaction() {
	// enqueue will fail if transaction with hash will be enqueued
	tx := common.CreateTransaction(context.Background(), common.SendTxArgs{})
	tx.Hash = gethcommon.Hash{1}
	s.Equal(ErrQueuedTxAlreadyProcessed, s.queue.Enqueue(tx))

	tx = common.CreateTransaction(context.Background(), common.SendTxArgs{})
	tx.Err = errors.New("error")
	s.Equal(ErrQueuedTxAlreadyProcessed, s.queue.Enqueue(tx))
}

func (s *QueueTestSuite) testDone(hash gethcommon.Hash, err error) *common.QueuedTx {
	tx := common.CreateTransaction(context.Background(), common.SendTxArgs{})
	s.NoError(s.queue.Enqueue(tx))
	s.NoError(s.queue.Done(tx.ID, hash, err))
	return tx
}

func (s *QueueTestSuite) TestDoneSuccess() {
	hash := gethcommon.Hash{1}
	tx := s.testDone(hash, nil)
	s.NoError(tx.Err)
	s.Equal(hash, tx.Hash)
	s.False(s.queue.Has(tx.ID))
	// event is sent only if transaction was removed from a queue
	select {
	case <-tx.Done:
	default:
		s.Fail("No event was sent to Done channel")
	}
}

func (s *QueueTestSuite) TestDoneTransientError() {
	hash := gethcommon.Hash{1}
	err := keystore.ErrDecrypt
	tx := s.testDone(hash, err)
	s.Equal(keystore.ErrDecrypt, tx.Err)
	s.Equal(gethcommon.Hash{}, tx.Hash)
	s.True(s.queue.Has(tx.ID))
}

func (s *QueueTestSuite) TestDoneError() {
	hash := gethcommon.Hash{1}
	err := errors.New("test")
	tx := s.testDone(hash, err)
	s.Equal(err, tx.Err)
	s.NotEqual(hash, tx.Hash)
	s.Equal(gethcommon.Hash{}, tx.Hash)
	s.False(s.queue.Has(tx.ID))
	// event is sent only if transaction was removed from a queue
	select {
	case <-tx.Done:
	default:
		s.Fail("No event was sent to Done channel")
	}
}

func (s QueueTestSuite) TestMultipleDone() {
	hash := gethcommon.Hash{1}
	err := keystore.ErrDecrypt
	tx := s.testDone(hash, err)
	s.NoError(s.queue.Done(tx.ID, hash, nil))
	s.Equal(ErrQueuedTxIDNotFound, s.queue.Done(tx.ID, hash, errors.New("timeout")))
}

func (s *QueueTestSuite) TestEviction() {
	var first *common.QueuedTx
	for i := 0; i < DefaultTxQueueCap; i++ {
		tx := common.CreateTransaction(context.Background(), common.SendTxArgs{})
		if first == nil {
			first = tx
		}
		s.NoError(s.queue.Enqueue(tx))
	}
	s.Equal(DefaultTxQueueCap, s.queue.Count())
	tx := common.CreateTransaction(context.Background(), common.SendTxArgs{})
	s.NoError(s.queue.Enqueue(tx))
	s.Equal(DefaultTxQueueCap, s.queue.Count())
	s.False(s.queue.Has(first.ID))
}
