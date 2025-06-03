package fees

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	baseFeesMainnet = []string{
		"0x6e0501af5", "0x6f1232450", "0x7b0c3bd93", "0x7657cebce", "0x700872e6e", "0x7785cda7e", "0x74145e5c7", "0x75d155d6e",
		"0x6e338bf0a", "0x6dc17226e", "0x6d1150d1c",
	}

	baseFeesMainnetBigIntSorted = []*big.Int{
		big.NewInt(29277621532), big.NewInt(29462307438), big.NewInt(29533149941), big.NewInt(29581950730), big.NewInt(29815415888),
		big.NewInt(30073630318), big.NewInt(31159870919), big.NewInt(31626452334), big.NewInt(31767456718), big.NewInt(32084122238),
		big.NewInt(33030389139),
	}

	baseFeesMainnetBigIntSortedUnique = baseFeesMainnetBigIntSorted

	priorityFeeMainnet = [][]string{
		{"0x30291a0"}, {"0x59682f00"}, {"0x59682f00"}, {"0x59682f00"}, {"0x30291a0"}, {"0x59682f00"}, {"0x3b9aca00"}, {"0x59682f00"},
		{"0x124f80"}, {"0x124f80"},
	}

	// nolint: unused
	priorityFeeMainnetBigIntSorted = []*big.Int{
		big.NewInt(1200000), big.NewInt(50500000), big.NewInt(50500000), big.NewInt(1000000000), big.NewInt(1500000000),
		big.NewInt(1500000000), big.NewInt(1500000000), big.NewInt(1500000000), big.NewInt(1500000000), big.NewInt(1500000000),
	}

	// nolint: unused
	priorityFeeMainnetBigIntSortedUnique = []*big.Int{
		big.NewInt(1200000), big.NewInt(50500000), big.NewInt(1000000000), big.NewInt(1500000000),
	}

	gasUsedRatioMainnet = []float64{
		0.5382303882944718, 0.931316908521827, 0.34705812379611983, 0.2867218121570287, 0.7674092750688878, 0.3847715,
		0.5598951666666667, 0.24141877992244926, 0.4838221092714951, 0.4749258055555556}

	baseFeesOptimism = []string{
		"0x11d", "0x11e", "0x11d", "0x11d", "0x11d", "0x11d", "0x11d", "0x11d", "0x11d", "0x11c", "0x11c", "0x11c", "0x11c",
		"0x11c", "0x11c", "0x11c", "0x11c", "0x11b", "0x11c", "0x11c", "0x11c", "0x11c", "0x11c", "0x11b", "0x11c", "0x11b",
		"0x11b", "0x11a", "0x11b", "0x11c", "0x11c", "0x11c", "0x11c", "0x11c", "0x11c", "0x11c", "0x11c", "0x11c", "0x11c",
		"0x11b", "0x11b", "0x11a", "0x11a", "0x11a", "0x11a", "0x11a", "0x11b", "0x11b", "0x11c", "0x11d", "0x11d",
	}

	baseFeesOptimismBigIntSorted = []*big.Int{
		big.NewInt(282), big.NewInt(282), big.NewInt(282), big.NewInt(282), big.NewInt(282), big.NewInt(282), big.NewInt(283),
		big.NewInt(283), big.NewInt(283), big.NewInt(283), big.NewInt(283), big.NewInt(283), big.NewInt(283), big.NewInt(283),
		big.NewInt(283), big.NewInt(284), big.NewInt(284), big.NewInt(284), big.NewInt(284), big.NewInt(284), big.NewInt(284),
		big.NewInt(284), big.NewInt(284), big.NewInt(284), big.NewInt(284), big.NewInt(284), big.NewInt(284), big.NewInt(284),
		big.NewInt(284), big.NewInt(284), big.NewInt(284), big.NewInt(284), big.NewInt(284), big.NewInt(284), big.NewInt(284),
		big.NewInt(284), big.NewInt(284), big.NewInt(284), big.NewInt(284), big.NewInt(284), big.NewInt(285), big.NewInt(285),
		big.NewInt(285), big.NewInt(285), big.NewInt(285), big.NewInt(285), big.NewInt(285), big.NewInt(285), big.NewInt(285),
		big.NewInt(285), big.NewInt(286),
	}

	baseFeesOptimismBigIntSortedUnique = []*big.Int{
		big.NewInt(282), big.NewInt(283), big.NewInt(284), big.NewInt(285), big.NewInt(286),
	}

	priorityFeeOptimism = [][]string{
		{"0xf4240"}, {"0xf423f"}, {"0xf6952"}, {"0xf4240"}, {"0xf4240"}, {"0xf4240"}, {"0xf4240"}, {"0xf4240"}, {"0xf4240"},
		{"0xf4240"}, {"0xf4240"}, {"0xf4240"}, {"0xf4240"}, {"0xf4240"}, {"0xf4240"}, {"0xf4240"}, {"0xf6952"}, {"0xf4241"},
		{"0xf4240"}, {"0xf4240"}, {"0xf4240"}, {"0xf4240"}, {"0x0"}, {"0xf4241"}, {"0xf4240"}, {"0xf4240"}, {"0xf6953"},
		{"0xf4242"}, {"0xf4240"}, {"0xf4240"}, {"0xf6950"}, {"0xf4240"}, {"0xf6952"}, {"0xf6952"}, {"0xf6952"}, {"0xf4240"},
		{"0xf4240"}, {"0xf6952"}, {"0xf4240"}, {"0xf4240"}, {"0xf4240"}, {"0xf4240"}, {"0xf4240"}, {"0xf6953"}, {"0xf6952"},
		{"0xf4240"}, {"0xf6951"}, {"0xf4240"}, {"0xf423f"}, {"0xf4240"},
	}

	// nolint: unused
	priorityFeeOptimismBigIntSorted = []*big.Int{
		big.NewInt(0), big.NewInt(999999), big.NewInt(999999), big.NewInt(1000000), big.NewInt(1000000), big.NewInt(1000000),
		big.NewInt(1000000), big.NewInt(1000000), big.NewInt(1000000), big.NewInt(1000000), big.NewInt(1000000), big.NewInt(1000000),
		big.NewInt(1000000), big.NewInt(1000000), big.NewInt(1000000), big.NewInt(1000000), big.NewInt(1000000), big.NewInt(1000000),
		big.NewInt(1000000), big.NewInt(1000000), big.NewInt(1000000), big.NewInt(1000000), big.NewInt(1000000), big.NewInt(1000000),
		big.NewInt(1000000), big.NewInt(1000000), big.NewInt(1000000), big.NewInt(1000000), big.NewInt(1000000), big.NewInt(1000000),
		big.NewInt(1000000), big.NewInt(1000000), big.NewInt(1000000), big.NewInt(1000000), big.NewInt(1000000), big.NewInt(1000000),
		big.NewInt(1000001), big.NewInt(1000001), big.NewInt(1000002), big.NewInt(1010000), big.NewInt(1010001), big.NewInt(1010002),
		big.NewInt(1010002), big.NewInt(1010002), big.NewInt(1010002), big.NewInt(1010002), big.NewInt(1010002), big.NewInt(1010002),
		big.NewInt(1010003), big.NewInt(1010003),
	}

	// nolint: unused
	priorityFeeOptimismBigIntSortedUnique = []*big.Int{
		big.NewInt(0), big.NewInt(999999), big.NewInt(1000000), big.NewInt(1000001), big.NewInt(1000002), big.NewInt(1010000),
		big.NewInt(1010001), big.NewInt(1010002), big.NewInt(1010003),
	}

	gasUsedRatioOptimism = []float64{
		0.17043076666666668, 0.005017633333333334, 0.024843566666666667, 0.09633871666666667, 0.07266241666666666,
		0.07642866666666667, 0.032275, 0.03833386666666667, 0.00419815, 0.08150265, 0.0555219, 0.12398543333333334, 0.0281475,
		0.07388538333333333, 0.16341473333333334, 0.11910195, 0.017414916666666665, 0.2674687, 0.09507331666666667, 0.14907565,
		0.10676838333333333, 0.06457223333333334, 0.00073065, 0.25560653333333333, 0.004907966666666667, 0.030404983333333333,
		0.006992933333333333, 0.3326633, 0.21239488333333334, 0.049208883333333335, 0.04813378333333333, 0.15355921666666666,
		0.06482011666666666, 0.06314621666666667, 0.05174276666666667, 0.0984776, 0.15978311666666667, 0.028946933333333334,
		0.012172633333333334, 0.08840158333333334, 0.006468466666666666, 0.14384575, 0.11492348333333334, 0.04476105, 0.0721188,
		0.2921225333333333, 0.06848103333333333, 0.18867503333333333, 0.2648652, 0.09275158333333333,
	}

	baseFeesArbitrum = []string{
		"0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100",
		"0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100",
		"0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100",
		"0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100",
		"0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100",
		"0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100", "0x5f5e100",
	}

	baseFeesArbitrumBigIntSorted = []*big.Int{
		big.NewInt(100000000), big.NewInt(100000000), big.NewInt(100000000), big.NewInt(100000000), big.NewInt(100000000),
		big.NewInt(100000000), big.NewInt(100000000), big.NewInt(100000000), big.NewInt(100000000), big.NewInt(100000000),
		big.NewInt(100000000), big.NewInt(100000000), big.NewInt(100000000), big.NewInt(100000000), big.NewInt(100000000),
		big.NewInt(100000000), big.NewInt(100000000), big.NewInt(100000000), big.NewInt(100000000), big.NewInt(100000000),
		big.NewInt(100000000), big.NewInt(100000000), big.NewInt(100000000), big.NewInt(100000000), big.NewInt(100000000),
		big.NewInt(100000000), big.NewInt(100000000), big.NewInt(100000000), big.NewInt(100000000), big.NewInt(100000000),
		big.NewInt(100000000), big.NewInt(100000000), big.NewInt(100000000), big.NewInt(100000000), big.NewInt(100000000),
		big.NewInt(100000000), big.NewInt(100000000), big.NewInt(100000000), big.NewInt(100000000), big.NewInt(100000000),
		big.NewInt(100000000), big.NewInt(100000000), big.NewInt(100000000), big.NewInt(100000000), big.NewInt(100000000),
		big.NewInt(100000000), big.NewInt(100000000), big.NewInt(100000000), big.NewInt(100000000), big.NewInt(100000000),
	}

	baseFeesArbitrumBigIntSortedUnique = []*big.Int{
		big.NewInt(100000000),
	}

	priorityFeeArbitrum = [][]string{
		{"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"},
		{"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"},
		{"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"}, {"0x0"},
	}

	// nolint: unused
	priorityFeeArbitrumBigIntSorted = []*big.Int{
		big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0),
		big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0),
		big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0),
		big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0),
		big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0),
		big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0),
		big.NewInt(0), big.NewInt(0),
	}

	// nolint: unused
	priorityFeeArbitrumBigIntSortedUnique = []*big.Int{
		big.NewInt(0),
	}

	gasUsedRatioArbitrum = []float64{
		1, 1, 1, 1, 1, 1, 1, 1, 0.0036689285714285712, 0.011981785714285714, 0.06032721428571428, 0.010431928571428571,
		0.01193192857142857, 0.021430714285714286, 0.025538357142857144, 0.014007357142857143, 0.016454857142857143,
		0.024767714285714285, 0.026267714285714287, 0.05319842857142857, 0.06342321428571429, 0.07247121428571429, 0.093301,
		0.01590142857142857, 0.027044285714285715, 0.028544285714285713, 0.03041642857142857, 0.029761214285714287, 0.0363775,
		0.006793, 0.015105857142857143, 0.04218464285714286, 0.06390228571428572, 0.0021190714285714285, 0.0036190714285714285,
		0.0051190714285714286, 0.008312857142857143, 0.009812857142857142, 0.0139205, 0.016300571428571428, 0.02461342857142857,
		0.003951785714285714, 0.0015, 0.050650071428571426, 0.1350182857142857, 0.14134221428571428, 0.14284221428571428, 0.00832,
		0.011673, 0.013545142857142856,
	}
)

func TestConvertToBigIntAndSort(t *testing.T) {

	tests := []struct {
		name          string
		feeHistory    *FeeHistory
		sortedBaseFee []*big.Int
		uniqueBaseFee []*big.Int
	}{
		{
			name: "baseFeesMainnet",
			feeHistory: &FeeHistory{
				BaseFeePerGas: baseFeesMainnet,
				Reward:        priorityFeeMainnet,
				GasUsedRatio:  gasUsedRatioMainnet,
			},
			sortedBaseFee: baseFeesMainnetBigIntSorted,
			uniqueBaseFee: baseFeesMainnetBigIntSortedUnique,
		},
		{
			name: "baseFeesOptimism",
			feeHistory: &FeeHistory{
				BaseFeePerGas: baseFeesOptimism,
				Reward:        priorityFeeOptimism,
				GasUsedRatio:  gasUsedRatioOptimism,
			},
			sortedBaseFee: baseFeesOptimismBigIntSorted,
			uniqueBaseFee: baseFeesOptimismBigIntSortedUnique,
		},
		{
			name: "baseFeesArbitrum",
			feeHistory: &FeeHistory{
				BaseFeePerGas: baseFeesArbitrum,
				Reward:        priorityFeeArbitrum,
				GasUsedRatio:  gasUsedRatioArbitrum,
			},
			sortedBaseFee: baseFeesArbitrumBigIntSorted,
			uniqueBaseFee: baseFeesArbitrumBigIntSortedUnique,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := convertToBigIntAndSort(test.feeHistory.BaseFeePerGas)
			assert.Equal(t, test.sortedBaseFee, result)

			result = removeDuplicatesFromSortedArray(result)
			assert.Equal(t, test.uniqueBaseFee, result)
		})
	}
}

func TestEstimatedTimeCalculation(t *testing.T) {

	tests := []struct {
		maxFee       *big.Int
		priorityFee  *big.Int
		expectedTime uint
	}{
		{
			maxFee:       big.NewInt(33767456719),
			priorityFee:  big.NewInt(1000000001),
			expectedTime: 15,
		},
		{
			maxFee:       big.NewInt(32760456719),
			priorityFee:  big.NewInt(1000000001),
			expectedTime: 15,
		},
		{
			maxFee:       big.NewInt(32067456719),
			priorityFee:  big.NewInt(1000000001),
			expectedTime: 25,
		},
	}

	feeHistory := &FeeHistory{
		BaseFeePerGas: baseFeesMainnet,
		Reward:        priorityFeeMainnet,
		GasUsedRatio:  gasUsedRatioMainnet,
	}

	for i, test := range tests {

		t.Run(fmt.Sprintf("test-%d", i), func(t *testing.T) {
			estimatedTime := estimatedTimeV2(feeHistory, test.maxFee, test.priorityFee, 1, 0)
			assert.Equal(t, test.expectedTime, estimatedTime)
		})
	}
}
