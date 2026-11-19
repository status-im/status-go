package rpc

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ErrorResponseSuite struct {
	suite.Suite
}

func TestErrorResponseSuite(t *testing.T) {
	suite.Run(t, new(ErrorResponseSuite))
}

func (s *ErrorResponseSuite) TestKeepsRequestID() {
	err := errors.New("there is no running node")
	s.Require().JSONEq(
		`{"jsonrpc":"2.0","id":7,"error":{"code":-32000,"message":"there is no running node"}}`,
		ErrorResponse(`{"jsonrpc":"2.0","id":7,"method":"wakuext_chats","params":[]}`, err),
	)
	s.Require().JSONEq(
		`{"jsonrpc":"2.0","id":"abc","error":{"code":-32000,"message":"there is no running node"}}`,
		ErrorResponse(`{"jsonrpc":"2.0","id":"abc","method":"wakuext_chats","params":[]}`, err),
	)
}

func (s *ErrorResponseSuite) TestNullIDForUnreadableRequest() {
	err := errors.New("there is no running node")
	s.Require().JSONEq(
		`{"jsonrpc":"2.0","id":null,"error":{"code":-32000,"message":"there is no running node"}}`,
		ErrorResponse(`not json`, err),
	)
	s.Require().JSONEq(
		`{"jsonrpc":"2.0","id":null,"error":{"code":-32000,"message":"there is no running node"}}`,
		ErrorResponse(`{"jsonrpc":"2.0","method":"wakuext_chats"}`, err),
	)
}
