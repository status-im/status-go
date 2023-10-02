package verification

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/t/helpers"
)

func TestPersistenceSuite(t *testing.T) {
	suite.Run(t, new(PersistenceSuite))
}

type PersistenceSuite struct {
	suite.Suite

	db *Persistence
}

func (s *PersistenceSuite) SetupTest() {
	db, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	s.Require().NoError(err)
	err = sqlite.Migrate(db)
	s.Require().NoError(err)

	s.db = &Persistence{db: db}
}

func (s *PersistenceSuite) TestVerificationRequests() {
	request := &Request{
		ID:            "0xabc",
		From:          "0x01",
		To:            "0x02",
		Challenge:     "ABC",
		Response:      "",
		RequestedAt:   uint64(time.Unix(1000, 0).Unix()),
		RepliedAt:     uint64(time.Unix(0, 0).Unix()),
		RequestStatus: RequestStatusPENDING,
	}

	// Test Insert
	err := s.db.SaveVerificationRequest(request)
	s.Require().NoError(err)

	// Test Found
	dbRequest, err := s.db.GetVerificationRequest("0xabc")
	s.Require().NoError(err)
	s.Require().Equal(request, dbRequest)

	// Test Not Found
	dbRequest2, err := s.db.GetVerificationRequest("0xdef")
	s.Require().NoError(err)
	s.Require().Nil(dbRequest2)

	// Test Accept
	err = s.db.AcceptContactVerificationRequest("0xabc", "XYZ")
	s.Require().NoError(err)

	dbRequest, err = s.db.GetVerificationRequest("0xabc")
	s.Require().NoError(err)
	s.Require().NotNil(dbRequest)
	s.Require().Equal(RequestStatusACCEPTED, dbRequest.RequestStatus)
	s.Require().Equal("XYZ", dbRequest.Response)
	s.Require().NotEqual(time.Unix(0, 0), dbRequest.RepliedAt)

	// Test Decline
	request = &Request{
		ID:            "0x01",
		From:          "0x03",
		To:            "0x02",
		Challenge:     "ABC",
		Response:      "",
		RequestedAt:   uint64(time.Unix(1000, 0).Unix()),
		RepliedAt:     uint64(time.Unix(0, 0).Unix()),
		RequestStatus: RequestStatusPENDING,
	}

	err = s.db.SaveVerificationRequest(request)
	s.NoError(err)

	err = s.db.DeclineContactVerificationRequest("0x01")
	s.NoError(err)

	dbRequest, err = s.db.GetVerificationRequest("0x01")
	s.NoError(err)
	s.Equal(RequestStatusDECLINED, dbRequest.RequestStatus)
	s.NotEqual(time.Unix(0, 0), dbRequest.RepliedAt)

}

func (s *PersistenceSuite) TestTrustStatus() {

	err := s.db.SetTrustStatus("0x01", TrustStatusTRUSTED, 1000)
	s.NoError(err)

	err = s.db.SetTrustStatus("0x02", TrustStatusUNKNOWN, 1001)
	s.NoError(err)

	trustStatus, err := s.db.GetTrustStatus("0x01")
	s.NoError(err)
	s.Equal(TrustStatusTRUSTED, trustStatus)

	err = s.db.SetTrustStatus("0x01", TrustStatusUNTRUSTWORTHY, 1000)
	s.NoError(err)

	trustStatus, err = s.db.GetTrustStatus("0x01")
	s.NoError(err)
	s.Equal(TrustStatusUNTRUSTWORTHY, trustStatus)

	trustStatus, err = s.db.GetTrustStatus("0x03")
	s.NoError(err)
	s.Equal(TrustStatusUNKNOWN, trustStatus)

	statuses, err := s.db.GetAllTrustStatus()
	s.NoError(err)

	s.Len(statuses, 2)
	s.Equal(TrustStatusUNTRUSTWORTHY, statuses["0x01"])
	s.Equal(TrustStatusUNKNOWN, statuses["0x02"])

	// Upsert
	success, err := s.db.UpsertTrustStatus("0x03", TrustStatusTRUSTED, 1000)
	s.NoError(err)
	s.True(success)

	trustStatus, err = s.db.GetTrustStatus("0x03")
	s.NoError(err)
	s.Equal(TrustStatusTRUSTED, trustStatus)

	success, err = s.db.UpsertTrustStatus("0x03", TrustStatusUNKNOWN, 500) // should not be successful without error, because of being older than latest value in the DB
	s.NoError(err)
	s.False(success)

	trustStatus, err = s.db.GetTrustStatus("0x03")
	s.NoError(err)
	s.Equal(TrustStatusTRUSTED, trustStatus)

	success, err = s.db.UpsertTrustStatus("0x03", TrustStatusUNKNOWN, 1500)
	s.NoError(err)
	s.False(success)

	trustStatus, err = s.db.GetTrustStatus("0x03")
	s.NoError(err)
	s.Equal(TrustStatusUNKNOWN, trustStatus)
}
