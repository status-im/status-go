package account

// Basic imports
import (
	"testing"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/suite"
)

type AccountUtilsTestSuite struct {
	suite.Suite
	validKey string
}

func (suite *AccountUtilsTestSuite) SetupTest() {
	suite.validKey = "0xF35E0325dad87e2661c4eF951d58727e6d583d5c"
}

func (suite *AccountUtilsTestSuite) TestToAddress() {
	addr := ToAddress(suite.validKey)
	suite.Equal(suite.validKey, addr.String())
}

func (suite *AccountUtilsTestSuite) TestToAddressInvalidAddress() {
	addr := ToAddress("foobar")
	suite.Nil(addr)
}

func (suite *AccountUtilsTestSuite) TestFromAddress() {
	var flagtests = []struct {
		in  string
		out string
	}{
		{suite.validKey, suite.validKey},
		{"foobar", "0x0000000000000000000000000000000000000000"},
	}

	for _, tt := range flagtests {
		addr := FromAddress(tt.in)
		suite.Equal(tt.out, addr.String())
	}
}

func (suite *AccountUtilsTestSuite) TestHex() {
	var addr *SelectedExtKey
	cr, _ := crypto.GenerateKey()
	var flagtests = []struct {
		in  *SelectedExtKey
		out string
	}{
		{&SelectedExtKey{
			Address:    FromAddress(suite.validKey),
			AccountKey: &keystore.Key{PrivateKey: cr},
		}, suite.validKey},
		{addr, "0x0"},
	}

	for _, tt := range flagtests {
		suite.Equal(tt.in.Hex(), tt.out)
	}
}

func TestAccountUtilsTestSuite(t *testing.T) {
	suite.Run(t, new(AccountUtilsTestSuite))
}
