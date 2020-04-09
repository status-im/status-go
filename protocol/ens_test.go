package protocol

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ENSSuite struct {
	suite.Suite
}

func TestENSSuite(t *testing.T) {
	suite.Run(t, new(ENSSuite))
}

func (s *ENSSuite) TestShouldBeVerified() {
	testCases := []struct {
		Name     string
		Contact  Contact
		Expected bool
		TimeNow  uint64
	}{
		{
			Name:     "valid eth name",
			Contact:  Contact{Name: "vitalik.eth"},
			Expected: true,
		},
		{
			Name: "valid eth name,some retries",
			Contact: Contact{
				Name:                   "vitalik.eth",
				ENSVerifiedAt:          10,
				ENSVerificationRetries: 4,
			},
			Expected: true,
			TimeNow:  10 + ENSBackoffTimeSec*4*16 + 1,
		},
		{
			Name:     "Empty name",
			Contact:  Contact{},
			Expected: false,
		},
		{
			Name:     "invalid eth name",
			Contact:  Contact{Name: "vitalik.eth2"},
			Expected: false,
		},
		{
			Name: "Already  verified",
			Contact: Contact{
				Name:        "vitalik.eth",
				ENSVerified: true,
			},
			Expected: false,
		},

		{
			Name: "verified recently",
			Contact: Contact{
				Name:                   "vitalik.eth",
				ENSVerifiedAt:          10,
				ENSVerificationRetries: 4,
			},
			Expected: false,
			TimeNow:  10 + ENSBackoffTimeSec*4*16 - 1,
		},
		{
			Name: "max retries reached",
			Contact: Contact{
				Name:                   "vitalik.eth",
				ENSVerifiedAt:          10,
				ENSVerificationRetries: 11,
			},
			Expected: false,
			TimeNow:  10 + ENSBackoffTimeSec*5*2048 + 1,
		},
	}
	for _, tc := range testCases {
		s.Run(tc.Name, func() {
			response := shouldENSBeVerified(&tc.Contact, tc.TimeNow)
			s.Equal(tc.Expected, response)
		})
	}
}

func (s *ENSSuite) TestHasENSNameChanged() {
	testCases := []struct {
		Name     string
		Contact  Contact
		NewName  string
		Clock    uint64
		Expected bool
	}{
		{
			Name:     "clock value is greater",
			Contact:  Contact{LastENSClockValue: 0},
			Clock:    1,
			NewName:  "vitalik.eth",
			Expected: true,
		},
		{
			Name:     "name in empty",
			Contact:  Contact{LastENSClockValue: 0},
			Clock:    1,
			Expected: false,
		},
		{
			Name:     "name is invalid",
			Contact:  Contact{LastENSClockValue: 0},
			Clock:    1,
			NewName:  "vitalik.eth2",
			Expected: false,
		},
		{
			Name: "name is identical",
			Contact: Contact{
				Name:              "vitalik.eth",
				LastENSClockValue: 0,
			},
			Clock:    1,
			NewName:  "vitalik.eth",
			Expected: false,
		},

		{
			Name:     "clock value is less",
			Contact:  Contact{LastENSClockValue: 1},
			Clock:    0,
			NewName:  "vitalik.eth",
			Expected: false,
		},
	}
	for _, tc := range testCases {
		s.Run(tc.Name, func() {
			response := hasENSNameChanged(&tc.Contact, tc.NewName, tc.Clock)
			s.Equal(tc.Expected, response)
		})
	}
}
