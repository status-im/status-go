package protocol

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/protocol/protobuf"
)

func TestMessengerStatusUpdatesSuite(t *testing.T) {
	suite.Run(t, new(MessengerStatusUpdatesSuite))
}

type MessengerStatusUpdatesSuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerStatusUpdatesSuite) TestNextHigherClockValueOfAutomaticStatusUpdates() {

	statusUpdate1 := UserStatus{
		StatusType: int(protobuf.StatusUpdate_AUTOMATIC),
		Clock:      100,
		CustomText: "",
		PublicKey:  "pub-key1",
	}

	err := s.m.persistence.InsertStatusUpdate(statusUpdate1)
	s.Require().NoError(err)

	statusUpdate2 := UserStatus{
		StatusType: int(protobuf.StatusUpdate_AUTOMATIC),
		Clock:      200,
		CustomText: "",
		PublicKey:  "pub-key2",
	}

	err = s.m.persistence.InsertStatusUpdate(statusUpdate2)
	s.Require().NoError(err)

	statusUpdate3 := UserStatus{
		StatusType: int(protobuf.StatusUpdate_AUTOMATIC),
		Clock:      300,
		CustomText: "",
		PublicKey:  "pub-key3",
	}

	err = s.m.persistence.InsertStatusUpdate(statusUpdate3)
	s.Require().NoError(err)

	// nextClock: clock value next higher than passed clock, of status update of type StatusUpdate_AUTOMATIC
	nextClock, err := s.m.persistence.NextHigherClockValueOfAutomaticStatusUpdates(100)
	s.Require().NoError(err)

	s.Require().Equal(nextClock, uint64(200))

}

func (s *MessengerStatusUpdatesSuite) TestDeactivatedStatusUpdates() {

	statusUpdate1 := UserStatus{
		StatusType: int(protobuf.StatusUpdate_AUTOMATIC),
		Clock:      100,
		CustomText: "",
		PublicKey:  "pub-key1",
	}

	err := s.m.persistence.InsertStatusUpdate(statusUpdate1)
	s.Require().NoError(err)

	statusUpdate2 := UserStatus{
		StatusType: int(protobuf.StatusUpdate_AUTOMATIC),
		Clock:      200,
		CustomText: "",
		PublicKey:  "pub-key2",
	}

	err = s.m.persistence.InsertStatusUpdate(statusUpdate2)
	s.Require().NoError(err)

	statusUpdate3 := UserStatus{
		StatusType: int(protobuf.StatusUpdate_AUTOMATIC),
		Clock:      400,
		CustomText: "",
		PublicKey:  "pub-key3",
	}

	err = s.m.persistence.InsertStatusUpdate(statusUpdate3)
	s.Require().NoError(err)

	statusUpdate4 := UserStatus{
		StatusType: int(protobuf.StatusUpdate_AUTOMATIC),
		Clock:      400, // Adding duplicate clock value for testing
		CustomText: "",
		PublicKey:  "pub-key4",
	}

	err = s.m.persistence.InsertStatusUpdate(statusUpdate4)
	s.Require().NoError(err)

	statusUpdate5 := UserStatus{
		StatusType: int(protobuf.StatusUpdate_AUTOMATIC),
		Clock:      500,
		CustomText: "",
		PublicKey:  "pub-key5",
	}

	err = s.m.persistence.InsertStatusUpdate(statusUpdate5)
	s.Require().NoError(err)

	// Lower limit is not included, but upper limit is included
	// So every status update in this range (lowerClock upperClock] will be deactivated
	deactivatedAutomaticStatusUpdates, err := s.m.persistence.DeactivatedAutomaticStatusUpdates(100, 400)
	s.Require().NoError(err)

	count := len(deactivatedAutomaticStatusUpdates)
	s.Require().Equal(3, count)

	// Status is deactivated
	s.Require().Equal(int(protobuf.StatusUpdate_INACTIVE), deactivatedAutomaticStatusUpdates[0].StatusType)

	// Lower range starts at 201 (clock + 1)
	// (clock is bumped, so that client replaces old status update with new one)
	s.Require().Equal(uint64(201), deactivatedAutomaticStatusUpdates[0].Clock)

	//Upper rannge ends at 401 (clock + 1)
	s.Require().Equal(uint64(401), deactivatedAutomaticStatusUpdates[count-1].Clock)
}
