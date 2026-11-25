package backend

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/internal/protocol/requests"
	"github.com/status-im/status-go/internal/signal"
	"github.com/status-im/status-go/internal/testutils"
)

type LoginStateSuite struct {
	suite.Suite

	backend *StatusBackend
	rootDir string

	mu                  sync.Mutex
	stateAtLoginSignals []LoginState
}

func TestLoginStateSuite(t *testing.T) {
	suite.Run(t, new(LoginStateSuite))
}

func (s *LoginStateSuite) SetupTest() {
	s.stateAtLoginSignals = nil
	s.rootDir = s.T().TempDir()
	s.backend = NewStatusBackend(testutils.MustCreateTestLogger())
	s.backend.UpdateRootDataDir(s.rootDir)
	initializeApplicationWithMediaServerPortHelper(s.T(), s.backend, reserveLocalhostPort(s.T()))

	signal.SetHandler(func(data []byte) {
		if !strings.Contains(string(data), signal.EventLoggedIn) {
			return
		}
		state := s.backend.LoginState()
		s.mu.Lock()
		defer s.mu.Unlock()
		s.stateAtLoginSignals = append(s.stateAtLoginSignals, state)
	})
}

func (s *LoginStateSuite) TearDownTest() {
	signal.ResetHandler()
	_ = s.backend.Logout()
}

func (s *LoginStateSuite) statesAtLoginSignals() []LoginState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]LoginState(nil), s.stateAtLoginSignals...)
}

func (s *LoginStateSuite) createAccountAndLogin() string {
	acc, err := s.backend.CreateAccountAndLogin(&requests.CreateAccount{
		DisplayName:        "login-state-test",
		CustomizationColor: "#ffffff",
		Password:           testPassword,
		RootDataDir:        s.rootDir,
		LogFilePath:        s.rootDir + "/log",
		KdfIterations:      1,
	})
	s.Require().NoError(err)
	return acc.KeyUID
}

func (s *LoginStateSuite) TestNoneBeforeAnyLogin() {
	state := s.backend.LoginState()
	s.Require().Equal(LoginStateNone, state.State)
	s.Require().Nil(state.Account)
}

func (s *LoginStateSuite) TestInProgressFromAcceptedRequestUntilLoggedIn() {
	s.backend.LoginRequestAccepted()
	s.Require().Equal(LoginStateInProgress, s.backend.LoginState().State)
	s.Require().Nil(s.backend.LoginState().Account)

	keyUID := s.createAccountAndLogin()

	state := s.backend.LoginState()
	s.Require().Equal(LoginStateLoggedIn, state.State)
	s.Require().NotNil(state.Account)
	s.Require().Equal(keyUID, state.Account.KeyUID)

	atSignal := s.statesAtLoginSignals()
	s.Require().Len(atSignal, 1)
	s.Require().Equal(LoginStateLoggedIn, atSignal[0].State, "node.login must not arrive before the state says logged in")
}

func (s *LoginStateSuite) TestLoginAccountGoesThroughInProgress() {
	keyUID := s.createAccountAndLogin()
	s.Require().NoError(s.backend.Logout())
	s.Require().Equal(LoginStateNone, s.backend.LoginState().State)

	s.backend.LoginRequestAccepted()
	s.Require().Equal(LoginStateInProgress, s.backend.LoginState().State)

	s.Require().NoError(s.backend.LoginAccount(&requests.Login{
		KeyUID:   keyUID,
		Password: testPassword,
	}))

	state := s.backend.LoginState()
	s.Require().Equal(LoginStateLoggedIn, state.State)
	s.Require().Equal(keyUID, state.Account.KeyUID)
}

func (s *LoginStateSuite) TestFailedLoginGoesBackToNone() {
	keyUID := s.createAccountAndLogin()
	s.Require().NoError(s.backend.Logout())

	s.backend.LoginRequestAccepted()
	_ = s.backend.LoginAccount(&requests.Login{
		KeyUID:   keyUID,
		Password: "wrong-password",
	})

	state := s.backend.LoginState()
	s.Require().Equal(LoginStateNone, state.State)
	s.Require().Nil(state.Account)

	atSignal := s.statesAtLoginSignals()
	s.Require().Len(atSignal, 2)
	s.Require().Equal(LoginStateNone, atSignal[1].State)
}

func (s *LoginStateSuite) TestLogoutGoesBackToNone() {
	s.createAccountAndLogin()
	s.Require().Equal(LoginStateLoggedIn, s.backend.LoginState().State)

	_, err := s.backend.LogoutAndAdvanceSignalEpoch()
	s.Require().NoError(err)

	state := s.backend.LoginState()
	s.Require().Equal(LoginStateNone, state.State)
	s.Require().Nil(state.Account)
}

func (s *LoginStateSuite) TestJSONContract() {
	none, err := json.Marshal(LoginState{State: LoginStateNone})
	s.Require().NoError(err)
	s.Require().JSONEq(`{"state":"none"}`, string(none))

	inProgress, err := json.Marshal(LoginState{State: LoginStateInProgress})
	s.Require().NoError(err)
	s.Require().JSONEq(`{"state":"inProgress"}`, string(inProgress))

	keyUID := s.createAccountAndLogin()
	loggedIn, err := json.Marshal(s.backend.LoginState())
	s.Require().NoError(err)

	var decoded struct {
		State   string                     `json:"state"`
		Account map[string]json.RawMessage `json:"account"`
	}
	s.Require().NoError(json.Unmarshal(loggedIn, &decoded))
	s.Require().Equal("loggedIn", decoded.State)
	s.Require().JSONEq(`"`+keyUID+`"`, string(decoded.Account["key-uid"]))

	active, err := s.backend.GetActiveAccount()
	s.Require().NoError(err)
	activeJSON, err := json.Marshal(active)
	s.Require().NoError(err)
	accountJSON, err := json.Marshal(decoded.Account)
	s.Require().NoError(err)
	s.Require().JSONEq(string(activeJSON), string(accountJSON), "account has the GetActiveAccount shape")
}
