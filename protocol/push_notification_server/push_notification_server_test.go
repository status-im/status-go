package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPushNotificationServerValidateRegistration(t *testing.T) {
	server := Server{}
	require.Equal(t, ErrEmptyPushNotificationRegisterMessage, server.ValidateRegistration(nil, nil))

}
