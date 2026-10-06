package router

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/pkg/services/wallet/requests"
	"github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor"
	mock_pathprocessor "github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor/mock"
	"github.com/status-im/status-go/pkg/services/wallet/router/sendtype"
)

// approvalPendingProcessor is a path processor that can name a gas limit while the
// approval its main tx needs is not mined yet.
type approvalPendingProcessor struct {
	*mock_pathprocessor.MockPathProcessor
	gas uint64
}

func (p approvalPendingProcessor) GasBeforeApproval(pathprocessor.ProcessorInputParams) (uint64, bool) {
	return p.gas, true
}

func TestEstimateMainTx(t *testing.T) {
	packed := []byte{0xab, 0xcd}

	testCases := []struct {
		name              string
		processorName     string
		sendType          sendtype.SendType
		approvalRequired  bool
		packErr           error
		estimateErr       error
		gasBeforeApproval uint64 // the processor names a stand-in when non-zero
		expectedData      []byte
		expectedGas       uint64
		expectErr         bool
	}{
		{"swap with approval estimates when it can", "Relay", sendtype.Swap, true, nil, nil, 0, packed, 150000, false},
		// the allowance isn't there yet, so a revert is expected; the fee stays unknown until the
		// path is re-evaluated after the approval is mined
		{"swap with approval tolerates an estimation failure", "Relay", sendtype.Swap, true, nil, errors.New("execution reverted"), 0, packed, 0, false},
		{"swap with approval tolerates a packing failure", "Relay", sendtype.Swap, true, errors.New("no quote"), nil, 0, nil, 0, false},
		// a cross-chain swap travels as a Bridge; the processor, not the send type, says it is a swap
		{"cross-chain swap with approval tolerates an estimation failure", "Relay", sendtype.Bridge, true, nil, errors.New("execution reverted"), 0, packed, 0, false},
		// ...and may name the gas to show until the approval is mined
		{"swap with approval shows the processor's gas before approval", "Relay", sendtype.Swap, true, nil, errors.New("execution reverted"), 123456, packed, 123456, false},
		{"cross-chain swap with approval shows the processor's gas before approval", "Relay", sendtype.Bridge, true, nil, errors.New("execution reverted"), 123456, packed, 123456, false},
		// the stand-in never replaces a failure the allowance does not explain
		{"swap without approval propagates errors", "Relay", sendtype.Swap, false, nil, errors.New("execution reverted"), 123456, nil, 0, true},
		{"transfer propagates errors", "Transfer", sendtype.Transfer, false, nil, errors.New("estimation failed"), 0, nil, 0, true},
		{"a non-swap processor with approval propagates errors", "Hop", sendtype.Bridge, true, nil, errors.New("estimation failed"), 0, nil, 0, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mock := mock_pathprocessor.NewMockPathProcessor(ctrl)
			mock.EXPECT().Name().Return(tc.processorName).AnyTimes()
			if tc.packErr != nil {
				mock.EXPECT().PackTxInputData(gomock.Any()).Return(nil, tc.packErr)
			} else {
				mock.EXPECT().PackTxInputData(gomock.Any()).Return(packed, nil)
				if tc.estimateErr != nil {
					mock.EXPECT().EstimateGas(gomock.Any(), packed).Return(uint64(0), tc.estimateErr)
				} else {
					mock.EXPECT().EstimateGas(gomock.Any(), packed).Return(uint64(150000), nil)
				}
			}
			var processor pathprocessor.PathProcessor = mock
			if tc.gasBeforeApproval != 0 {
				processor = approvalPendingProcessor{MockPathProcessor: mock, gas: tc.gasBeforeApproval}
			}

			r := &Router{logger: logutils.ZapLogger().Named("router-test")}
			input := &requests.RouteInputParams{SendType: tc.sendType}

			data, gas, err := r.estimateMainTx(input, processor, pathprocessor.ProcessorInputParams{}, tc.approvalRequired)
			if tc.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expectedData, data)
			require.Equal(t, tc.expectedGas, gas)
		})
	}
}
