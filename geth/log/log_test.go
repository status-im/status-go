package log_test

import (
	"errors"
	"os"
	"testing"

	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/log/providers/jsonfile"
	"github.com/stretchr/testify/suite"
)

func TestJailTestSuite(t *testing.T) {
	suite.Run(t, new(LogTestSuite))
}

type LogTestSuite struct {
	suite.Suite
}

func (s LogTestSuite) TestBatchWriter() {
	require := s.Require()

	bm := log.BatchEmit(1, func(entries []log.Entry) error {
		for _, en := range entries {
			if en.Level == log.RedAlertLvl {
				return errors.New("RedAlert")
			}
		}

		return nil
	})

	defer bm.Close()

	err := bm.Emit(log.WithMessage(log.InfoLvl, "Batch emitting data"))
	require.NoError(err)

	err = bm.Emit(log.WithMessage(log.RedAlertLvl, "Batch emitting data at critical").With("name", "thunder"))
	require.Error(err, "Expect error to occur")
}

func (s LogTestSuite) TestJSONFile() {
	require := s.Require()

	fileName := "log.hjson"

	defer os.RemoveAll(fileName)
	defer os.RemoveAll(fileName + "_1")

	jsm := jsonfile.JSONFile(fileName, ".", 10, 2)

	go func() {
		defer jsm.Close()

		err := jsm.Emit(log.WithMessage(log.InfoLvl, "Batch emitting data"))
		require.NoError(err)

		err = jsm.Emit(log.WithMessage(log.RedAlertLvl, "Batch emitting data at critical").With("name", "thunder"))
		require.NoError(err)

		err = jsm.Emit(log.WithMessage(log.InfoLvl, "Batch emitting info data").With("name", "thunder"))
		require.NoError(err)

		err = jsm.Emit(log.WithMessage(log.ErrorLvl, "Batch emitting error data").With("name", "thunder"))
		require.NoError(err)
	}()

	jsm.Wait()

	_, err := os.Stat(fileName)
	require.NoError(err)

	_, err = os.Stat(fileName + "_1")
	require.NoError(err)
}
