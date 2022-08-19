package rln

type MessageValidationResult int

const (
	MessageValidationResult_Unknown MessageValidationResult = iota
	MessageValidationResult_Valid
	MessageValidationResult_Invalid
	MessageValidationResult_Spam
)
