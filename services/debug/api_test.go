package debug

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
)

var (
	errShhPost = errors.New("shh post failed")
)

func TestDebugSuite(t *testing.T) {
	suite.Run(t, new(DebugSuite))
}

type DebugSuite struct {
	suite.Suite
	p    *MockPoster
	api  *PublicAPI
	ctx  context.Context
	msg  whisper.NewMessage
	hash hexutil.Bytes
}

func (s *DebugSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.p = NewMockPoster(ctrl)
	w := whisper.New(nil)
	service := New(w)
	service.SetPoster(s.p)

	symID, err := w.GenerateSymKey()
	s.NoError(err)
	s.msg = whisper.NewMessage{
		SymKeyID:  symID,
		PowTarget: whisper.DefaultMinimumPoW,
		PowTime:   200,
		Topic:     whisper.TopicType{0x01, 0x01, 0x01, 0x01},
		Payload:   []byte("hello"),
	}
	s.hash = []byte("hash")
	s.ctx = context.Background()

	s.api = NewAPI(service)
}

func (s *DebugSuite) TestPostconfirmErrors() {
	var testCases = []struct {
		name                string
		prepareExpectations func(*DebugSuite)
		expectedError       error
		expectedHash        hexutil.Bytes
	}{
		{
			name: "post errored",
			prepareExpectations: func(s *DebugSuite) {
				s.p.EXPECT().Post(s.ctx, s.msg).Return(nil, errShhPost)
			},
			expectedError: errShhPost,
			expectedHash:  nil,
		},
		{
			name: "post timeout error",
			prepareExpectations: func(s *DebugSuite) {
				postTimeout = 0 * time.Second
				s.p.EXPECT().Post(s.ctx, s.msg).Return(s.hash, nil)
			},
			expectedError: errTimeout,
			expectedHash:  s.hash,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.prepareExpectations(s)

			hash, err := s.api.PostSync(s.ctx, s.msg)

			s.Equal(tc.expectedError, err)
			s.Equal(tc.expectedHash, hash)
		})
	}
}
