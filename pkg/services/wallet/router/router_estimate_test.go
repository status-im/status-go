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

func TestEstimateMainTx(t *testing.T) {
	packed := []byte{0xab, 0xcd}

	testCases := []struct {
		name             string
		sendType         sendtype.SendType
		approvalRequired bool
		packErr          error
		estimateErr      error
		expectedData     []byte
		expectedGas      uint64
		expectErr        bool
	}{
		{"swap with approval estimates when it can", sendtype.Swap, true, nil, nil, packed, 150000, false},
		// the allowance isn't there yet, so a revert is expected; the fee stays unknown until the
		// path is re-evaluated after the approval is mined
		{"swap with approval tolerates an estimation failure", sendtype.Swap, true, nil, errors.New("execution reverted"), packed, 0, false},
		{"swap with approval tolerates a packing failure", sendtype.Swap, true, errors.New("no quote"), nil, nil, 0, false},
		{"swap without approval propagates errors", sendtype.Swap, false, nil, errors.New("execution reverted"), nil, 0, true},
		{"transfer propagates errors", sendtype.Transfer, false, nil, errors.New("estimation failed"), nil, 0, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			processor := mock_pathprocessor.NewMockPathProcessor(ctrl)
			processor.EXPECT().Name().Return("Relay").AnyTimes()
			if tc.packErr != nil {
				processor.EXPECT().PackTxInputData(gomock.Any()).Return(nil, tc.packErr)
			} else {
				processor.EXPECT().PackTxInputData(gomock.Any()).Return(packed, nil)
				if tc.estimateErr != nil {
					processor.EXPECT().EstimateGas(gomock.Any(), packed).Return(uint64(0), tc.estimateErr)
				} else {
					processor.EXPECT().EstimateGas(gomock.Any(), packed).Return(uint64(150000), nil)
				}
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
