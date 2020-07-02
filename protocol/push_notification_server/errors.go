package push_notification_server

import "errors"

var ErrInvalidPushNotificationRegistrationVersion = errors.New("invalid version")
var ErrEmptyPushNotificationRegistrationPayload = errors.New("empty payload")
var ErrMalformedPushNotificationRegistrationInstallationID = errors.New("invalid installationID")
var ErrEmptyPushNotificationRegistrationPublicKey = errors.New("no public key")
var ErrCouldNotUnmarshalPushNotificationRegistration = errors.New("could not unmarshal preferences")
var ErrInvalidCiphertextLength = errors.New("invalid cyphertext length")
var ErrMalformedPushNotificationRegistrationDeviceToken = errors.New("invalid device token")
var ErrMalformedPushNotificationRegistrationAccessToken = errors.New("invalid access token")
var ErrUnknownPushNotificationRegistrationTokenType = errors.New("invalid token type")
