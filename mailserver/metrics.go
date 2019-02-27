package mailserver

import "github.com/ethereum/go-ethereum/metrics"

var (
	// By default go-ethereum/metrics creates dummy metrics that don't register anything.
	// Real metrics are collected only if -metrics flag is set
	requestProcessTimer            = metrics.NewRegisteredTimer("mailserver/requestProcessTime", nil)
	requestProcessNetTimer         = metrics.NewRegisteredTimer("mailserver/requestProcessNetTime", nil)
	requestsMeter                  = metrics.NewRegisteredMeter("mailserver/requests", nil)
	requestsBatchedCounter         = metrics.NewRegisteredCounter("mailserver/requestsBatched", nil)
	requestErrorsCounter           = metrics.NewRegisteredCounter("mailserver/requestErrors", nil)
	sentEnvelopesMeter             = metrics.NewRegisteredMeter("mailserver/sentEnvelopes", nil)
	sentEnvelopesSizeMeter         = metrics.NewRegisteredMeter("mailserver/sentEnvelopesSize", nil)
	archivedMeter                  = metrics.NewRegisteredMeter("mailserver/archivedEnvelopes", nil)
	archivedSizeMeter              = metrics.NewRegisteredMeter("mailserver/archivedEnvelopesSize", nil)
	archivedErrorsCounter          = metrics.NewRegisteredCounter("mailserver/archiveErrors", nil)
	requestValidationErrorsCounter = metrics.NewRegisteredCounter("mailserver/requestValidationErrors", nil)
	processRequestErrorsCounter    = metrics.NewRegisteredCounter("mailserver/processRequestErrors", nil)
	historicResponseErrorsCounter  = metrics.NewRegisteredCounter("mailserver/historicResponseErrors", nil)
	syncRequestsMeter              = metrics.NewRegisteredMeter("mailserver/syncRequests", nil)
	deliverMailTimer               = metrics.NewRegisteredTimer("mailserver/deliverMailTime", nil)
)
