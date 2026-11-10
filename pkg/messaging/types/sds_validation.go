package types

// SDSMessageDisposition controls whether an SDS envelope reaches reliability processing.
type SDSMessageDisposition uint8

const (
	// SDSMessageDispositionReject discards the envelope before processing its SDS metadata.
	SDSMessageDispositionReject SDSMessageDisposition = iota
	// SDSMessageDispositionProcess allows normal SDS reliability processing.
	SDSMessageDispositionProcess
	// SDSMessageDispositionApplicationOnly unwraps content without processing SDS metadata.
	SDSMessageDispositionApplicationOnly
)
