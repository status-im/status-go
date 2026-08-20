package communities

import (
	"database/sql"
	"time"
)

func (s *CommunitySuite) TestRequestToJoin_Empty() {
	// Brand new RequestToJoin should be empty
	rtj := new(RequestToJoin)
	s.True(rtj.Empty(), "The RequestToJoin should be empty")

	// Add some values, should not be empty
	rtj.State = RequestToJoinStateAccepted
	rtj.Clock = uint64(time.Now().Unix())
	s.False(rtj.Empty(), "The RequestToJoin should not be empty")

	// Overwrite with a new RequestToJoin, should be empty
	rtj = new(RequestToJoin)
	s.True(rtj.Empty(), "The RequestToJoin should be empty")

	// Add some empty values, should be empty
	rtj.ChatID = ""
	rtj.ENSName = ""
	rtj.PublicKey = ""
	rtj.Clock = uint64(sql.NullInt64{}.Int64)
	rtj = new(RequestToJoin)
	s.True(rtj.Empty(), "The RequestToJoin should be empty")

	// Add some not empty values, should be not empty
	rtj.ChatID = "0x1234abcd"
	rtj.ENSName = "@samyoul"
	rtj.PublicKey = "0xfedc0987"
	s.False(rtj.Empty(), "The RequestToJoin should not be empty")
}
