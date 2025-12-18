package errors

const (
	ErrorCategory1 ErrorCategory = "1"
	ErrorCategory2 ErrorCategory = "2"
	ErrorCategory3 ErrorCategory = "3"
	ErrorCategory4 ErrorCategory = "4"
)

const (
	ErrCode1 ErrorCode = iota + 1
	ErrCode2
	ErrCode3
	ErrCode4
)

// Common errors
func Err3(arg string, arg2 string) *AccountsError {
	return NewError(ErrCode3, "error 3 message", getErrorCategoryTest).WithContext("arg", arg).WithContext("arg2", arg2)
}

func Err4(err error) *AccountsError {
	return WrapError(ErrCode4, "error 4 message", err, getErrorCategoryTest)
}

func getErrorCategoryTest(code ErrorCode) ErrorCategory {
	switch code {
	case ErrCode1:
		return ErrorCategory1
	case ErrCode2:
		return ErrorCategory2
	case ErrCode3:
		return ErrorCategory3
	case ErrCode4:
		return ErrorCategory4
	}
	return ErrorCategoryUnknown
}
