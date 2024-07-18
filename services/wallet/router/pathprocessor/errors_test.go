package pathprocessor

import (
	"context"
	"errors"
	"testing"

	s_errors "github.com/status-im/status-go/errors"

	"github.com/stretchr/testify/require"
)

func TestPlainError(t *testing.T) {
	const errString = "plain error"
	err := errors.New(errString)

	processorNames := []string{
		ProcessorTransferName,
		ProcessorTransferName,
		ProcessorBridgeHopName,
		ProcessorBridgeCelerName,
		ProcessorSwapParaswapName,
		ProcessorERC721Name,
		ProcessorERC1155Name,
		ProcessorENSRegisterName,
		ProcessorENSReleaseName,
		ProcessorENSPublicKeyName,
		ProcessorStickersBuyName,
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
	ppErrResp := createErrorResponse(ProcessorTransferName, err)

	castPPErrResp := ppErrResp.(*s_errors.ErrorResponse)
	require.Equal(t, errResp.Code, castPPErrResp.Code)
	require.Equal(t, errResp.Details, castPPErrResp.Details)
}
