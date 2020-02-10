package protocol

import (
	"math"
	"strings"
)

// maxRetries is the maximum number of attempts we do before giving up
const maxRetries uint64 = 11

// ENSBackoffTimeMs is the step of the exponential backoff
// we retry roughly for 17 hours after receiving the message 2^11 * 30000
const ENSBackoffTimeMs uint64 = 30000

// We calculate if it's too early to retry, by exponentially backing off
func verifiedENSRecentlyEnough(now, verifiedAt, retries uint64) bool {
	return now < verifiedAt+ENSBackoffTimeMs*retries*uint64(math.Exp2(float64(retries)))
}

func shouldENSBeVerified(c *Contact, now uint64) bool {
	if c.Name == "" {
		return false
	}

	if c.ENSVerified {
		return false
	}

	if c.ENSVerificationRetries >= maxRetries {
		return false
	}

	if verifiedENSRecentlyEnough(now, c.ENSVerifiedAt, c.ENSVerificationRetries) {
		return false
	}

	if !strings.HasSuffix(c.Name, ".eth") {
		return false
	}

	return true
}

// This should trigger re-verification of the ENS name for this contact
func hasENSNameChanged(c *Contact, newName string, clockValue uint64) bool {
	if c.LastENSClockValue > clockValue {
		return false
	}

	if newName == "" {
		return false
	}

	if !strings.HasSuffix(newName, ".eth") {
		return false
	}

	if newName == c.Name {
		return false
	}

	return true
}
