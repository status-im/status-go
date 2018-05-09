package accounts

import (
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	sdk "github.com/status-im/status-go-sdk"
	"github.com/status-im/status-go/geth/api"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/t/e2e"
	"github.com/stretchr/testify/suite"
)

func TestDummySDKTestSuite(t *testing.T) {
	suite.Run(t, new(DummySDKTestSuite))
}

type DummySDKTestSuite struct {
	e2e.BackendTestSuite
}

func (s *DummySDKTestSuite) nodeOptions() e2e.TestNodeOption {
	return e2e.TestNodeOption(func(config *params.NodeConfig) {
		config.StatusServiceEnabled = true

		config.WhisperConfig.Enabled = true
		config.WhisperConfig.EnableMailServer = false
		config.WhisperConfig.LightClient = true
		config.WhisperConfig.MinimumPoW = params.WhisperMinimumPoW
		config.WhisperConfig.TTL = params.WhisperTTL

		// TODO(adriacidre) remove this as shouldn't be needed since introduction of loadNodeConfig
		config.UpstreamConfig.Enabled = true
		config.UpstreamConfig.URL = "https://ropsten.infura.io/z6GCTmjdP3FETEJmMBI4"
		config.NetworkID = 4
		config.ClusterConfig.BootNodes = append(config.ClusterConfig.BootNodes, "enode://7ab298cedc4185a894d21d8a4615262ec6bdce66c9b6783878258e0d5b31013d30c9038932432f70e5b2b6a5cd323bf820554fcb22fbc7b45367889522e9c449@51.15.63.93:30303")
		config.ClusterConfig.BootNodes = append(config.ClusterConfig.BootNodes, "enode://f59e8701f18c79c5cbc7618dc7bb928d44dc2f5405c7d693dad97da2d8585975942ec6fd36d3fe608bfdc7270a34a4dd00f38cfe96b2baa24f7cd0ac28d382a1@51.15.79.88:30303")
		config.ClusterConfig.BootNodes = append(config.ClusterConfig.BootNodes, "enode://e2a3587b7b41acfc49eddea9229281905d252efba0baf565cf6276df17faf04801b7879eead757da8b5be13b05f25e775ab6d857ff264bc53a89c027a657dd10@51.15.45.114:30303")
		config.ClusterConfig.BootNodes = append(config.ClusterConfig.BootNodes, "enode://fe991752c4ceab8b90608fbf16d89a5f7d6d1825647d4981569ebcece1b243b2000420a5db721e214231c7a6da3543fa821185c706cbd9b9be651494ec97f56a@51.15.67.119:30303")
		config.ClusterConfig.BootNodes = append(config.ClusterConfig.BootNodes, "enode://482484b9198530ee2e00db89791823244ca41dcd372242e2e1297dd06f6d8dd357603960c5ad9cc8dc15fcdf0e4edd06b7ad7db590e67a0b54f798c26581ebd7@51.15.75.138:30303")
		config.ClusterConfig.BootNodes = append(config.ClusterConfig.BootNodes, "enode://9e99e183b5c71d51deb16e6b42ac9c26c75cfc95fff9dfae828b871b348354cbecf196dff4dd43567b26c8241b2b979cb4ea9f8dae2d9aacf86649dafe19a39a@51.15.79.176:30303")
		config.ClusterConfig.BootNodes = append(config.ClusterConfig.BootNodes, "enode://12d52c3796700fb5acff2c7d96df7bbb6d7109b67f3442ee3d99ac1c197016cddb4c3568bbeba05d39145c59c990cd64f76bc9b00d4b13f10095c49507dd4cf9@51.15.63.110:30303")
		config.ClusterConfig.BootNodes = append(config.ClusterConfig.BootNodes, "enode://0f7c65277f916ff4379fe520b875082a56e587eb3ce1c1567d9ff94206bdb05ba167c52272f20f634cd1ebdec5d9dfeb393018bfde1595d8e64a717c8b46692f@51.15.54.150:30303")
		config.ClusterConfig.BootNodes = append(config.ClusterConfig.BootNodes, "enode://e006f0b2dc98e757468b67173295519e9b6d5ff4842772acb18fd055c620727ab23766c95b8ee1008dea9e8ef61e83b1515ddb3fb56dbfb9dbf1f463552a7c9f@212.47.237.127:30303")
		config.ClusterConfig.BootNodes = append(config.ClusterConfig.BootNodes, "enode://d40871fc3e11b2649700978e06acd68a24af54e603d4333faecb70926ca7df93baa0b7bf4e927fcad9a7c1c07f9b325b22f6d1730e728314d0e4e6523e5cebc2@51.15.132.235:30303")
		config.ClusterConfig.BootNodes = append(config.ClusterConfig.BootNodes, "enode://ea37c9724762be7f668e15d3dc955562529ab4f01bd7951f0b3c1960b75ecba45e8c3bb3c8ebe6a7504d9a40dd99a562b13629cc8e5e12153451765f9a12a61d@163.172.189.205:30303")
		config.ClusterConfig.BootNodes = append(config.ClusterConfig.BootNodes, "enode://88c2b24429a6f7683fbfd06874ae3f1e7c8b4a5ffb846e77c705ba02e2543789d66fc032b6606a8d8888eb6239a2abe5897ce83f78dcdcfcb027d6ea69aa6fe9@163.172.157.61:30303")
		config.ClusterConfig.BootNodes = append(config.ClusterConfig.BootNodes, "enode://ce6854c2c77a8800fcc12600206c344b8053bb90ee3ba280e6c4f18f3141cdc5ee80bcc3bdb24cbc0e96dffd4b38d7b57546ed528c00af6cd604ab65c4d528f6@163.172.153.124:30303")
		config.ClusterConfig.BootNodes = append(config.ClusterConfig.BootNodes, "enode://00ae60771d9815daba35766d463a82a7b360b3a80e35ab2e0daa25bdc6ca6213ff4c8348025e7e1a908a8f58411a364fe02a0fb3c2aa32008304f063d8aaf1a2@163.172.132.85:30303")
		config.ClusterConfig.BootNodes = append(config.ClusterConfig.BootNodes, "enode://86ebc843aa51669e08e27400e435f957918e39dc540b021a2f3291ab776c88bbda3d97631639219b6e77e375ab7944222c47713bdeb3251b25779ce743a39d70@212.47.254.155:30303")
		config.ClusterConfig.BootNodes = append(config.ClusterConfig.BootNodes, "enode://a1ef9ba5550d5fac27f7cbd4e8d20a643ad75596f307c91cd6e7f85b548b8a6bf215cca436d6ee436d6135f9fe51398f8dd4c0bd6c6a0c332ccb41880f33ec12@51.15.218.125:30303")
		config.DataDir = "/tmp/testnet_rpc"

		config.LightEthConfig.Enabled = false

	})
}

func (s *DummySDKTestSuite) TestPubSub() {
	s.StartTestBackend(s.nodeOptions())
	defer s.StopTestBackend()

	client := sdk.New(newRPCClient(s.Backend))
	ac, err := client.SignupAndLogin("111222333")
	if err != nil {
		s.Fail("Could not sign up and login")
	}

	ch, err := ac.JoinPublicChannel("supu")
	if err != nil {
		s.Fail("Could not join a public channel")
	}

	ch.Subscribe(func(m *sdk.Msg) {
		if m.Type == sdk.PNBroadcastAvailabilityType {
			props := m.Properties.(*sdk.PNBroadcastAvailabilityMsg)
			println("-------")
			spew.Dump(props)
			println("-------")
		}
	})

	ch.PNBroadcastAvailabilityRequest()

	if s.Backend.StatusNode().GethNode() != nil {
		s.Backend.StatusNode().GethNode().Wait()
	}

	time.Sleep(time.Second * 10)
	s.Backend.StatusNode().GethNode().Stop()

}

type RPCClient struct {
	b *api.StatusBackend
}

func newRPCClient(b *api.StatusBackend) *RPCClient {
	return &RPCClient{b: b}
}

func (c *RPCClient) Call(request interface{}) (response interface{}, err error) {
	response = c.b.CallPrivateRPC(request.(string))
	return
}
