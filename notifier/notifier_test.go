package notifier

import (
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"
)

type NotifierTestSuite struct {
	suite.Suite
	url string
	n   *Notifier
}

func (s *NotifierTestSuite) SetupTest() {
	s.url = "http://localhost:3000"
	s.n = New(s.url)
}

var flagtests = []struct {
	response string
	err      error
}{
	{
		response: `{"counts":1,"logs":[{"type":"failed-push","platform":"android","token":"1111111","message":"Hello world","error":"invalid registration token"}],"success":"ok"}`,
		err:      errors.New("invalid registration token"),
	},
	{
		response: `{"counts":1,"success":"ok"}`,
		err:      nil,
	},
}

func (s *NotifierTestSuite) TestSendNotifications() {
	for _, tt := range flagtests {
		var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(tt.response)); err != nil {
				log.Println(err.Error())
			}
		}))
		defer apiStub.Close()

		s.n.url = apiStub.URL
		notification := Notification{
			Tokens:   []string{"t1", "t2"},
			Platform: 2,
			Message:  "hello world",
		}

		err := s.n.Send([]*Notification{&notification})
		s.Equal(tt.err, err)
	}
}

func TestNotifierTestSuite(t *testing.T) {
	suite.Run(t, new(NotifierTestSuite))
}
