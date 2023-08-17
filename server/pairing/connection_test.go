package pairing

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/server"
	"github.com/status-im/status-go/server/servertest"
)

const (
	connectionStringV1 = "cs2:4FHRnp:Q4:uqnnMwVUfJc2Fkcaojet8F1ufKC3hZdGEt47joyBx9yd:BbnZ7Gc66t54a9kEFCf7FW8SGQuYypwHVeNkRYeNoqV6"
	connectionStringV2 = "cs3:kDDauj5:Q4:uqnnMwVUfJc2Fkcaojet8F1ufKC3hZdGEt47joyBx9yd:BbnZ7Gc66t54a9kEFCf7FW8SGQuYypwHVeNkRYeNoqV6"
	port               = 1337
)

func TestConnectionParamsSuite(t *testing.T) {
	suite.Run(t, new(ConnectionParamsSuite))
}

type ConnectionParamsSuite struct {
	suite.Suite
	servertest.TestKeyComponents
	servertest.TestCertComponents
	servertest.TestLoggerComponents

	server *BaseServer
}

func (s *ConnectionParamsSuite) SetupSuite() {
	s.SetupKeyComponents(s.T())
	s.SetupCertComponents(s.T())
	s.SetupLoggerComponents()

	ip := server.LocalHostIP
	ips := []net.IP{ip}

	cert, _, err := GenerateCertFromKey(s.PK, s.NotBefore, ips, []string{})
	s.Require().NoError(err)

	sc := ServerConfig{
		PK:          &s.PK.PublicKey,
		EK:          s.AES,
		Cert:        &cert,
		IPAddresses: ips,
		ListenIP:    net.IPv4zero,
	}

	bs := server.NewServer(&cert, net.IPv4zero.String(), nil, s.Logger)
	err = bs.SetPort(port)
	s.Require().NoError(err)

	s.server = &BaseServer{
		Server: bs,
		config: sc,
	}
}

func (s *ConnectionParamsSuite) TestConnectionParams_ToString() {
	cp, err := s.server.MakeConnectionParams()
	s.Require().NoError(err)

	cps := cp.ToString()
	s.Require().Equal(connectionStringV2, cps)
}

func (s *ConnectionParamsSuite) TestConnectionParams_Generate() {

	testCases := []struct {
		description string
		cs          string
	}{
		{description: "ConnectionString_version1", cs: connectionStringV1},
		{description: "ConnectionString_version2", cs: connectionStringV2},
	}

	for _, tc := range testCases {
		s.T().Run(tc.description, func(t *testing.T) {
			cp := new(ConnectionParams)
			err := cp.FromString(connectionStringV2)
			s.Require().NoError(err)

			u, err := cp.URL(0)
			s.Require().NoError(err)

			expectedURL := fmt.Sprintf("https://%s:%d", server.LocalHostIP.String(), port)

			s.Require().Equal(expectedURL, u.String())
			s.Require().Equal(server.LocalHostIP.String(), u.Hostname())
			s.Require().Equal(strconv.Itoa(port), u.Port())

			s.Require().True(cp.publicKey.Equal(&s.PK.PublicKey))
			s.Require().Equal(s.AES, cp.aesKey)
		})
	}
}

func (s *ConnectionParamsSuite) TestConnectionParams_ParseNetIps() {

	in := []net.IP{
		{192, 168, 1, 42},
		net.ParseIP("fe80::6fd7:5ce4:554f:165a"),
		{172, 16, 9, 1},
		net.ParseIP("fe80::ffa5:98e1:285c:42eb"),
		net.ParseIP("fe80::c1f:ee0d:1476:dd9a"),
	}
	bytes := SerializeNetIps(in)

	s.Require().Equal(bytes,
		[]byte{
			2,               // v4 count
			192, 168, 1, 42, // v4 1
			172, 16, 9, 1, // v4 2
			3,                                                                            // v6 count
			0xfe, 0x80, 0, 0, 0, 0, 0, 0, 0x6f, 0xd7, 0x5c, 0xe4, 0x55, 0x4f, 0x16, 0x5a, // v6 1
			0xfe, 0x80, 0, 0, 0, 0, 0, 0, 0xff, 0xa5, 0x98, 0xe1, 0x28, 0x5c, 0x42, 0xeb, // v6 2
			0xfe, 0x80, 0, 0, 0, 0, 0, 0, 0x0c, 0x1f, 0xee, 0x0d, 0x14, 0x76, 0xdd, 0x9a, // v6 3
		})

	out, err := ParseNetIps(bytes)

	s.Require().NoError(err)
	s.Require().Len(in, 5)

	sort.SliceStable(in, func(i, j int) bool {
		return in[i].String() < in[j].String()
	})
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].String() < out[j].String()
	})

	s.Require().Equal(in, out)
}
