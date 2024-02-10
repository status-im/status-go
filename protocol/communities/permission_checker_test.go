package communities

import (
	"testing"

	"github.com/stretchr/testify/suite"

	gethcommon "github.com/ethereum/go-ethereum/common"
)

func TestPermissionCheckerSuite(t *testing.T) {
	suite.Run(t, new(PermissionCheckerSuite))
}

type PermissionCheckerSuite struct {
	suite.Suite
}

func (s *PermissionCheckerSuite) TestMergeValidCombinations() {

	permissionChecker := DefaultPermissionChecker{}

	combination1 := &AccountChainIDsCombination{
		Address:  gethcommon.HexToAddress("0xA"),
		ChainIDs: []uint64{1},
	}

	combination2 := &AccountChainIDsCombination{
		Address:  gethcommon.HexToAddress("0xB"),
		ChainIDs: []uint64{5},
	}

	combination3 := &AccountChainIDsCombination{
		Address:  gethcommon.HexToAddress("0xA"),
		ChainIDs: []uint64{5},
	}

	combination4 := &AccountChainIDsCombination{
		Address:  gethcommon.HexToAddress("0xB"),
		ChainIDs: []uint64{5},
	}

	mergedCombination := permissionChecker.MergeValidCombinations([]*AccountChainIDsCombination{combination1, combination2},
		[]*AccountChainIDsCombination{combination3, combination4})

	s.Require().Len(mergedCombination, 2)
	chains1 := mergedCombination[0].ChainIDs
	chains2 := mergedCombination[1].ChainIDs

	if len(chains1) == 2 {
		s.Equal([]uint64{1, 5}, chains1)
		s.Equal([]uint64{5}, chains2)
	} else {
		s.Equal([]uint64{1, 5}, chains2)
		s.Equal([]uint64{5}, chains1)
	}

}
