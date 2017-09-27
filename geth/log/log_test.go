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
	infoCache        []log.Entry
	errCache         []log.Entry
	redAlertCache    []log.Entry
	yellowAlertCache []log.Entry
}

func (s *LogTestSuite) SetupSuite() {
	log.InitMetric(log.New(
		log.FilterLevelWith(log.InfoLvl, func(en log.Entry) error {
			s.infoCache = append(s.infoCache, en)
			return nil
		}),
		log.FilterLevelWith(log.ErrorLvl, func(en log.Entry) error {
			s.errCache = append(s.errCache, en)
			return nil
		}),
		log.FilterLevelWith(log.RedAlertLvl, func(en log.Entry) error {
			s.redAlertCache = append(s.redAlertCache, en)
			return nil
		}),
		log.FilterLevelWith(log.YellowAlertLvl, func(en log.Entry) error {
			s.yellowAlertCache = append(s.yellowAlertCache, en)
			return nil
		}),
	))
}

func (s *LogTestSuite) TearDownSuite() {
	log.InitMetric(nil)
	s.infoCache = nil
	s.errCache = nil
	s.yellowAlertCache = nil
	s.redAlertCache = nil
}

func (s *LogTestSuite) TestLogger() {
	require := s.Require()

	err := log.Send(log.WithMessage(log.InfoLvl, "Batch emitting data"))
	require.NoError(err)

	err = log.Send(log.WithMessage(log.RedAlertLvl, "Batch emitting data at critical").With("name", "thunder"))
	require.NoError(err)

	err = log.Send(log.WithMessage(log.YellowAlertLvl, "Batch emitting info data").With("name", "thunder"))
	require.NoError(err)

	err = log.Send(log.WithMessage(log.ErrorLvl, "Batch emitting error data").With("name", "thunder"))
	require.NoError(err)

	require.Len(s.infoCache, 1, "Should have 1 entry in info")
	require.Len(s.errCache, 1, "Should have 1 entry in error")
	require.Len(s.redAlertCache, 1, "Should have 1 entry in redAlert")
	require.Len(s.yellowAlertCache, 1, "Should have 1 entry in yellowAlert")
}

func (s *LogTestSuite) TestBatchWriter() {
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

func (s *LogTestSuite) TestJSONFile() {
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
