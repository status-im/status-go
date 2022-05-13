package verification

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/protocol/sqlite"
)

func TestPersistenceSuite(t *testing.T) {
	suite.Run(t, new(PersistenceSuite))
}

type PersistenceSuite struct {
	suite.Suite

	db *Persistence
}

func (s *PersistenceSuite) SetupTest() {
	s.db = nil

	dbPath, err := ioutil.TempFile("", "")
	s.NoError(err, "creating temp file for db")

	db, err := sqlite.Open(dbPath.Name(), "")
	s.NoError(err, "creating sqlite db instance")

	s.db = &Persistence{db: db}
}

func (s *PersistenceSuite) TestVerificationRequests() {
	request := &Request{
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
	s.NoError(err)

	// Test Found
	dbRequest, err := s.db.GetVerificationRequestFrom("0x01")
	s.NoError(err)
	s.Equal(request, dbRequest)

	// Test Not Found
	dbRequest2, err := s.db.GetVerificationRequestFrom("0x02")
	s.NoError(err)
	s.Nil(dbRequest2)

	// Test Found
	dbRequest3, err := s.db.GetVerificationRequestSentTo("0x02")
	s.NoError(err)
	s.Equal(request, dbRequest3)

	// Test Accept
	err = s.db.AcceptContactVerificationRequest("0x01", "XYZ")
	s.NoError(err)

	dbRequest, err = s.db.GetVerificationRequestFrom("0x01")
	s.NoError(err)
	s.Equal(RequestStatusACCEPTED, dbRequest.RequestStatus)
	s.Equal("XYZ", dbRequest.Response)
	s.NotEqual(time.Unix(0, 0), dbRequest.RepliedAt)

	// Test Decline
	request = &Request{
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

	err = s.db.DeclineContactVerificationRequest("0x03")
	s.NoError(err)

	dbRequest, err = s.db.GetVerificationRequestFrom("0x03")
	s.NoError(err)
	s.Equal(RequestStatusDECLINED, dbRequest.RequestStatus)
	s.NotEqual(time.Unix(0, 0), dbRequest.RepliedAt)

	// Test Upsert fails because of older record
	ogDbRequestStatus := dbRequest.RequestStatus
	dbRequest.RequestStatus = RequestStatusPENDING
	shouldSync, err := s.db.UpsertVerificationRequest(dbRequest)
	s.NoError(err)
	s.False(shouldSync)

	dbRequest.RequestStatus = ogDbRequestStatus
	dbRequest2, err = s.db.GetVerificationRequestFrom("0x03")
	s.NoError(err)
	s.Equal(dbRequest, dbRequest2)

	// Test upsert success (update)
	dbRequest2.RequestStatus = RequestStatusPENDING
	dbRequest2.RepliedAt = dbRequest2.RepliedAt + 100
	shouldSync, err = s.db.UpsertVerificationRequest(dbRequest2)
	s.NoError(err)
	s.True(shouldSync)

	// Test upsert success (insert)
	verifReq := &Request{
		From:          "0x0A",
		To:            "0x0B",
		Challenge:     "123",
		Response:      "456",
		RequestedAt:   uint64(time.Now().Unix()),
		RepliedAt:     uint64(time.Now().Unix()),
		RequestStatus: RequestStatusPENDING,
	}

	shouldSync, err = s.db.UpsertVerificationRequest(verifReq)
	s.NoError(err)
	s.True(shouldSync)

	dbRequest, err = s.db.GetVerificationRequestFrom("0x0A")
	s.NoError(err)
	s.Equal(verifReq.To, dbRequest.To)
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
