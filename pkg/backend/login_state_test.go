package backend

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/internal/db/multiaccounts"
	"github.com/status-im/status-go/internal/protocol/requests"
	"github.com/status-im/status-go/internal/signal"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/params"
)

type nodeLoginSeen struct {
	Seq          uint64
	Error        string
	StateAtEvent LoginState
}

type LoginStateSuite struct {
	suite.Suite

	backend *StatusBackend
	rootDir string

	mu         sync.Mutex
	nodeLogins []nodeLoginSeen
}

func TestLoginStateSuite(t *testing.T) {
	suite.Run(t, new(LoginStateSuite))
}

func (s *LoginStateSuite) SetupTest() {
	s.nodeLogins = nil
	s.rootDir = s.T().TempDir()
	s.backend = NewStatusBackend(testutils.MustCreateTestLogger())
	s.backend.UpdateRootDataDir(s.rootDir)
	initializeApplicationWithMediaServerPortHelper(s.T(), s.backend, reserveLocalhostPort(s.T()))

	signal.SetHandler(func(data []byte) {
		if !strings.Contains(string(data), `"type":"`+signal.EventLoggedIn+`"`) {
			return
		}
		var envelope struct {
			Event struct {
				Seq   uint64 `json:"seq"`
				Error string `json:"error"`
			} `json:"event"`
		}
		s.Require().NoError(json.Unmarshal(data, &envelope))
		state := s.backend.LoginState()
		s.mu.Lock()
		defer s.mu.Unlock()
		s.nodeLogins = append(s.nodeLogins, nodeLoginSeen{
			Seq:          envelope.Event.Seq,
			Error:        envelope.Event.Error,
			StateAtEvent: state,
		})
	})
}

func (s *LoginStateSuite) TearDownTest() {
	signal.ResetHandler()
	_ = s.backend.Logout()
}

func (s *LoginStateSuite) seenNodeLogins() []nodeLoginSeen {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]nodeLoginSeen(nil), s.nodeLogins...)
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

func (s *LoginStateSuite) TestNoneWithSeqZeroBeforeAnyLogin() {
	state := s.backend.LoginState()
	s.Require().Equal(LoginStateNone, state.State)
	s.Require().Equal(uint64(0), state.Seq)
	s.Require().Nil(state.Account)
}

func (s *LoginStateSuite) TestLoggedInWithItsSeqBeforeNodeLogin() {
	keyUID := s.createAccountAndLogin()

	state := s.backend.LoginState()
	s.Require().Equal(LoginStateLoggedIn, state.State)
	s.Require().Equal(uint64(1), state.Seq)
	s.Require().NotNil(state.Account)
	s.Require().Equal(keyUID, state.Account.KeyUID)

	seen := s.seenNodeLogins()
	s.Require().Len(seen, 1)
	s.Require().Equal(uint64(1), seen[0].Seq, "node.login carries the seq of its login")
	s.Require().Equal(LoginStateLoggedIn, seen[0].StateAtEvent.State, "node.login must not arrive before the state says logged in")
}

func (s *LoginStateSuite) TestEachLoginGetsTheNextSeq() {
	keyUID := s.createAccountAndLogin()
	s.Require().NoError(s.backend.Logout())

	s.Require().NoError(s.backend.LoginAccount(&requests.Login{
		KeyUID:   keyUID,
		Password: testPassword,
	}))

	state := s.backend.LoginState()
	s.Require().Equal(LoginStateLoggedIn, state.State)
	s.Require().Equal(uint64(2), state.Seq)
	s.Require().Equal(keyUID, state.Account.KeyUID)

	seen := s.seenNodeLogins()
	s.Require().Len(seen, 2)
	s.Require().Equal(uint64(2), seen[1].Seq)
}

func (s *LoginStateSuite) TestFailedLoginGoesBackToNoneKeepingItsSeq() {
	keyUID := s.createAccountAndLogin()
	s.Require().NoError(s.backend.Logout())

	_ = s.backend.LoginAccount(&requests.Login{
		KeyUID:   keyUID,
		Password: "wrong-password",
	})

	state := s.backend.LoginState()
	s.Require().Equal(LoginStateNone, state.State)
	s.Require().Equal(uint64(2), state.Seq)
	s.Require().Nil(state.Account)

	seen := s.seenNodeLogins()
	s.Require().Len(seen, 2)
	s.Require().NotEmpty(seen[1].Error)
	s.Require().Equal(uint64(2), seen[1].Seq)
	s.Require().Equal(LoginStateNone, seen[1].StateAtEvent.State)
}

func (s *LoginStateSuite) TestLogoutGoesBackToNone() {
	s.createAccountAndLogin()
	s.Require().Equal(LoginStateLoggedIn, s.backend.loginState.get())

	_, err := s.backend.LogoutAndAdvanceSignalEpoch()
	s.Require().NoError(err)

	s.Require().Equal(LoginStateNone, s.backend.loginState.get())
	s.Require().Equal(uint64(1), s.backend.LoginState().Seq)
}

func (s *LoginStateSuite) TestPostStartFailureSendsNodeLoginWithTheError() {
	seq, err := s.backend.AcceptLoginRequest()
	s.Require().NoError(err)
	s.Require().Equal(uint64(1), seq)

	s.Require().Error(s.backend.LoggedIn("0xnot-opened", nil))

	s.Require().Equal(LoginStateNone, s.backend.LoginState().State)
	seen := s.seenNodeLogins()
	s.Require().Len(seen, 1)
	s.Require().NotEmpty(seen[0].Error)
	s.Require().Equal(uint64(1), seen[0].Seq)
}

func (s *LoginStateSuite) TestStartNodeWithKeyEarlyFailureSendsNodeLoginWithTheError() {
	acc := multiaccounts.Account{KeyUID: "0x1234", KDFIterations: 1}

	s.Require().Error(s.backend.StartNodeWithKey(acc, testPassword, "not-a-hex-key", &params.NodeConfig{}))

	s.Require().Equal(LoginStateNone, s.backend.LoginState().State)
	seen := s.seenNodeLogins()
	s.Require().Len(seen, 1)
	s.Require().NotEmpty(seen[0].Error)
	s.Require().Equal(uint64(1), seen[0].Seq)
}

func (s *LoginStateSuite) TestLoginRequestRejectedWhileLoggedIn() {
	s.createAccountAndLogin()

	_, err := s.backend.AcceptLoginRequest()
	s.Require().ErrorIs(err, ErrAlreadyLoggedIn)

	state := s.backend.LoginState()
	s.Require().Equal(LoginStateLoggedIn, state.State)
	s.Require().Equal(uint64(1), state.Seq)
}

func (s *LoginStateSuite) TestJSONContract() {
	none, err := json.Marshal(LoginState{State: LoginStateNone})
	s.Require().NoError(err)
	s.Require().JSONEq(`{"state":"none","seq":0}`, string(none))

	inProgress, err := json.Marshal(LoginState{State: LoginStateInProgress, Seq: 3})
	s.Require().NoError(err)
	s.Require().JSONEq(`{"state":"inProgress","seq":3}`, string(inProgress))

	keyUID := s.createAccountAndLogin()
	loggedIn, err := json.Marshal(s.backend.LoginState())
	s.Require().NoError(err)

	var decoded struct {
		State   string                     `json:"state"`
		Seq     uint64                     `json:"seq"`
		Account map[string]json.RawMessage `json:"account"`
	}
	s.Require().NoError(json.Unmarshal(loggedIn, &decoded))
	s.Require().Equal("loggedIn", decoded.State)
	s.Require().Equal(uint64(1), decoded.Seq)
	s.Require().JSONEq(`"`+keyUID+`"`, string(decoded.Account["key-uid"]))

	active, err := s.backend.GetActiveAccount()
	s.Require().NoError(err)
	activeJSON, err := json.Marshal(active)
	s.Require().NoError(err)
	accountJSON, err := json.Marshal(decoded.Account)
	s.Require().NoError(err)
	s.Require().JSONEq(string(activeJSON), string(accountJSON), "account has the GetActiveAccount shape")
}
