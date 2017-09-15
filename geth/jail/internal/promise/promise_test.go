package promise_test

import (
	"testing"
	"time"

	"github.com/robertkrimen/otto"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/geth/jail/internal/loop"
	"github.com/status-im/status-go/geth/jail/internal/promise"
	"github.com/status-im/status-go/geth/jail/internal/vm"
)

func (s *PromiseSuite) TestResolve() {
	err := s.vm.Set("__resolve", func(str string) {
		defer func() { s.ch <- struct{}{} }()

		s.Equal("good", str)
	})
	s.NoError(err)

	err = s.loop.Eval(`
		var p = new Promise(function(resolve, reject) {
			setTimeout(function() {
				resolve('good');
			}, 10);
		});

		p.then(function(d) {
			__resolve(d);
		});

		p.catch(function(err) {
			throw err;
		});
	`)
	s.NoError(err)

	select {
	case <-s.ch:
	case <-time.After(1 * time.Second):
		s.Fail("test timed out")
		return
	}
}

func (s *PromiseSuite) TestReject() {
	err := s.vm.Set("__reject", func(str string) {
		defer func() { s.ch <- struct{}{} }()

		s.Equal("bad", str)
	})
	s.NoError(err)

	err = s.loop.Eval(`
		var p = new Promise(function(resolve, reject) {
			setTimeout(function() {
				reject('bad');
			}, 10);
		});

		p.catch(function(err) {
			__reject(err);
		});
	`)
	s.NoError(err)

	select {
	case <-s.ch:
	case <-time.After(1 * time.Second):
		s.Fail("test timed out")
		return
	}
}

type PromiseSuite struct {
	suite.Suite

	loop *loop.Loop
	vm   *vm.VM

	ch chan struct{}
}

func (s *PromiseSuite) SetupTest() {
	o := otto.New()
	s.vm = vm.New(o)
	s.loop = loop.New(s.vm)

	go s.loop.Run()

	err := promise.Define(s.vm, s.loop)
	s.NoError(err)

	s.ch = make(chan struct{})
}

func TestPromiseSuite(t *testing.T) {
	suite.Run(t, new(PromiseSuite))
}
