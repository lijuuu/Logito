package dlq

const (
	FailureTypeParseError      = "parseError"
	FailureTypeDBFailure       = "dbFailure"
	FailureTypeValidationError = "validationError"
	FailureTypeTimeoutError    = "timeoutError"
)
