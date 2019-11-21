package whisperv6

import "github.com/ethereum/go-ethereum/metrics"

var (
	envelopeAddedCounter           = metrics.NewRegisteredCounter("whisper/envelopeAdded", nil)
	envelopeNewAddedCounter        = metrics.NewRegisteredCounter("whisper/envelopeNewAdded", nil)
	envelopeClearedCounter         = metrics.NewRegisteredCounter("whisper/envelopeCleared", nil)
	envelopeErrFromFutureCounter   = metrics.NewRegisteredCounter("whisper/envelopeErrFromFuture", nil)
	envelopeErrVeryOldCounter      = metrics.NewRegisteredCounter("whisper/envelopeErrVeryOld", nil)
	envelopeErrExpiredCounter      = metrics.NewRegisteredCounter("whisper/envelopeErrExpired", nil)
	envelopeErrOversizedCounter    = metrics.NewRegisteredCounter("whisper/envelopeErrOversized", nil)
	envelopeErrLowPowCounter       = metrics.NewRegisteredCounter("whisper/envelopeErrLowPow", nil)
	envelopeErrNoBloomMatchCounter = metrics.NewRegisteredCounter("whisper/envelopeErrNoBloomMatch", nil)
	envelopeSizeMeter              = metrics.NewRegisteredMeter("whisper/envelopeSize", nil)
)
