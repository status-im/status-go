package pathprocessor

import (
	"context"
	"errors"
	"testing"

	s_errors "github.com/status-im/status-go/errors"

	walletCommon "github.com/status-im/status-go/services/wallet/common"

	"github.com/stretchr/testify/require"
)

func TestPlainError(t *testing.T) {
	const errString = "plain error"
	err := errors.New(errString)

	processorNames := []string{
		walletCommon.ProcessorTransferName,
		walletCommon.ProcessorTransferName,
		walletCommon.ProcessorBridgeHopName,
		walletCommon.ProcessorBridgeCelerName,
		walletCommon.ProcessorSwapParaswapName,
		walletCommon.ProcessorERC721Name,
		walletCommon.ProcessorERC1155Name,
		walletCommon.ProcessorENSRegisterName,
		walletCommon.ProcessorENSReleaseName,
		walletCommon.ProcessorENSPublicKeyName,
		walletCommon.ProcessorStickersBuyName,
	}

	for _, processorName := range processorNames {
		ppErrResp := createErrorResponse(processorName, err)

		castPPErrResp := ppErrResp.(*s_errors.ErrorResponse)
		require.NotEqual(t, s_errors.GenericErrorCode, castPPErrResp.Code)
		require.Equal(t, errString, castPPErrResp.Details)
	}
}

func TestContextErrors(t *testing.T) {
	ppErrResp := createErrorResponse("Unknown", context.Canceled)
	require.Equal(t, ErrContextCancelled, ppErrResp)

	ppErrResp = createErrorResponse("Unknown", context.DeadlineExceeded)
	require.Equal(t, ErrContextDeadlineExceeded, ppErrResp)
}

func TestErrorResponse(t *testing.T) {
	const errString = "error response"
	err := errors.New(errString)
	errResp := s_errors.CreateErrorResponseFromError(err)
	ppErrResp := createErrorResponse("Unknown", errResp)

	castPPErrResp := ppErrResp.(*s_errors.ErrorResponse)
	require.Equal(t, s_errors.GenericErrorCode, castPPErrResp.Code)
	require.Equal(t, errString, castPPErrResp.Details)
}

func TestNonGenericErrorResponse(t *testing.T) {
	errResp := &s_errors.ErrorResponse{
		Code:    "Not Generic Code",
		Details: "Not Generic Error Response",
	}
	err := s_errors.CreateErrorResponseFromError(errResp)
	ppErrResp := createErrorResponse(walletCommon.ProcessorTransferName, err)

	castPPErrResp := ppErrResp.(*s_errors.ErrorResponse)
	require.Equal(t, errResp.Code, castPPErrResp.Code)
	require.Equal(t, errResp.Details, castPPErrResp.Details)
}

func TestCustomErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "no error - nil",
			err:      nil,
			expected: false,
		},
		{
			name:     "not error response error",
			err:      errors.New("unknown error"),
			expected: false,
		},
		{
			name:     "not custom error",
			err:      ErrFromChainNotSupported,
			expected: false,
		},
		{
			name:     "custom error",
			err:      ErrTransferCustomError,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, IsCustomError(tt.err))
		})
	}
}
