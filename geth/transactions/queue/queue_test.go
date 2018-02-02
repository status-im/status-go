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
	enquedTx, err := s.queue.Get(tx.ID)
	s.NoError(err)
	s.NoError(s.queue.LockInprogress(tx.ID))
	s.Equal(tx, enquedTx)

	// verify that tx was marked as being inprogress
	s.Equal(ErrQueuedTxInProgress, s.queue.LockInprogress(tx.ID))
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

func (s *QueueTestSuite) TestAlreadyEnqueued() {
	tx := common.CreateTransaction(context.Background(), common.SendTxArgs{})
	s.NoError(s.queue.Enqueue(tx))
	s.Equal(ErrQueuedTxExist, s.queue.Enqueue(tx))
	// try to enqueue another tx to double check locking
	tx = common.CreateTransaction(context.Background(), common.SendTxArgs{})
	s.NoError(s.queue.Enqueue(tx))
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
	// event is sent only if transaction was removed from a queue
	select {
	case rst := <-tx.Result:
		s.NoError(rst.Error)
		s.Equal(hash, rst.Hash)
		s.False(s.queue.Has(tx.ID))
	default:
		s.Fail("No event was sent to Done channel")
	}
}

func (s *QueueTestSuite) TestDoneTransientError() {
	hash := gethcommon.Hash{1}
	err := keystore.ErrDecrypt
	tx := s.testDone(hash, err)
	s.True(s.queue.Has(tx.ID))
	_, inp := s.queue.inprogress[tx.ID]
	s.False(inp)
}

func (s *QueueTestSuite) TestDoneError() {
	hash := gethcommon.Hash{1}
	err := errors.New("test")
	tx := s.testDone(hash, err)
	// event is sent only if transaction was removed from a queue
	select {
	case rst := <-tx.Result:
		s.Equal(err, rst.Error)
		s.NotEqual(hash, rst.Hash)
		s.Equal(gethcommon.Hash{}, rst.Hash)
		s.False(s.queue.Has(tx.ID))
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
