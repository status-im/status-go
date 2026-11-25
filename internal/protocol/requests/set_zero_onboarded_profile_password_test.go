package requests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetZeroOnboardedProfilePasswordValidate(t *testing.T) {
	require.ErrorIs(t, (&SetZeroOnboardedProfilePassword{NewPassword: "0xpass"}).Validate(), ErrSetZeroOnboardedProfilePasswordInvalidKeyUID)
	require.ErrorIs(t, (&SetZeroOnboardedProfilePassword{KeyUID: "0xkey"}).Validate(), ErrSetZeroOnboardedProfilePasswordInvalidNewPassword)
	require.NoError(t, (&SetZeroOnboardedProfilePassword{KeyUID: "0xkey", NewPassword: "0xpass"}).Validate())
}
