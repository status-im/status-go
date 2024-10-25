package common

import (
	"testing"
)

var testCases = []struct {
	errorString string
	errorTypes  []DialErrorType
}{
	{
		errorString: "failed to dial: failed to dial 16Uiu2HAmNFvubdwLtyScgQKMVL7Ppwvd7RZskgThtPAGqMrUfs1V: all dials failed\n  * [/ip4/0.0.0.0/tcp/55136] dial tcp4 0.0.0.0:60183->146.4.106.194:55136: i/o timeout",
		errorTypes:  []DialErrorType{ErrorIOTimeout},
	},
	{
		errorString: "failed to dial: failed to dial 16Uiu2HAmC1BsqZfy9exnA3DiQHAo3gdAopTQRErLUjK8WoospTwq: all dials failed\n  * [/ip4/0.0.0.0/tcp/46949] dial tcp4 0.0.0.0:60183->0.0.0.0:46949: i/o timeout\n  * [/ip4/0.0.0.0/tcp/51063] dial tcp4 0.0.0.0:60183->0.0.0.0:51063: i/o timeout",
		errorTypes:  []DialErrorType{ErrorIOTimeout, ErrorIOTimeout},
	},
	{
		errorString: "failed to dial: failed ito dial 16Uiu2HAkyjvXPmymR5eRnvxCufRGZdfRrgjME6bmn3Xo6aprE1eo: all dials failed\n  * [/ip4/0.0.0.0/tcp/443/wss/p2p/16Uiu2HAmB7Ur9HQqo3cWDPovRQjo57fxWWDaQx27WxSzDGhN4JKg/p2p-circuit] error opening relay circuit: CONNECTION_FAILED (203)\n  * [/ip4/0.0.0.0/tcp/30303/p2p/16Uiu2HAmB7Ur9HQqo3cWDPovRQjo57fxWWDaQx27WxSzDGhN4JKg/p2p-circuit] concurrent active dial through the same relay failed with a protocol error\n  * [/ip4/0.0.0.0/tcp/30303/p2p/16Uiu2HAmAUdrQ3uwzuE4Gy4D56hX6uLKEeerJAnhKEHZ3DxF1EfT/p2p-circuit] error opening relay circuit: CONNECTION_FAILED (203)\n  * [/ip4/0.0.0.0/tcp/443/wss/p2p/16Uiu2HAmAUdrQ3uwzuE4Gy4D56hX6uLKEeerJAnhKEHZ3DxF1EfT/p2p-circuit] concurrent active dial through the same relay failed with a protocol error",
		errorTypes:  []DialErrorType{ErrorRelayCircuitFailed, ErrorConcurrentDialFailed, ErrorRelayCircuitFailed, ErrorConcurrentDialFailed},
	},
	{
		errorString: "failed to dial: failed to dial 16Uiu2HAm9QijC9d2GsGKPLLF7cZXMFEadqvN7FqhFJ2z5jdW6AFY: all dials failed\n  * [/ip4/0.0.0.0/tcp/64012] dial tcp4 0.0.0.0:64012: connect: connection refused",
		errorTypes:  []DialErrorType{ErrorConnectionRefused},
	},
	{
		errorString: "failed to dial: failed to dial 16Uiu2HAm7jXmopqB6BUJAQH1PKcZULfSKgj9rC9pyBRKwJGTiRHf: all dials failed\n  * [/ip4/34.135.13.87/tcp/30303/p2p/16Uiu2HAm8mUZ18tBWPXDQsaF7PbCKYA35z7WB2xNZH2EVq1qS8LJ/p2p-circuit] error opening relay circuit: NO_RESERVATION (204)\n  * [/ip4/34.170.192.39/tcp/30303/p2p/16Uiu2HAmMELCo218hncCtTvC2Dwbej3rbyHQcR8erXNnKGei7WPZ/p2p-circuit] error opening relay circuit: NO_RESERVATION (204)\n  * [/ip4/178.72.78.116/tcp/42841] dial tcp4 0.0.0.0:60183->178.72.78.116:42841: i/o timeout",
		errorTypes:  []DialErrorType{ErrorRelayNoReservation, ErrorRelayNoReservation, ErrorIOTimeout},
	},
	{
		errorString: "failed to dial: failed to dial 16Uiu2HAmMUYpufreYsUBo4A56BQDnbMwN4mhP3wMWTM4reS8ivxd: all dials failed\n  * [/ip4/0.0.0.0/tcp/52957] unknown",
		errorTypes:  []DialErrorType{ErrorUnknown},
	},
}

func TestParseDialErrors(t *testing.T) {
	for _, testCase := range testCases {
		parsedErrors := ParseDialErrors(testCase.errorString)
		for i, err := range parsedErrors {
			if err.ErrType != testCase.errorTypes[i] {
				t.Errorf("Expected error type %v, got %v", testCase.errorTypes[i], err.ErrType)
			}
		}
	}
}
