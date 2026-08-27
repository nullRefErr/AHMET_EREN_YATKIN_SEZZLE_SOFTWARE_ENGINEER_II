package domain

import "errors"

// Sentinel errors returned by Calculate. Callers compare them with errors.Is;
// the HTTP layer is the only place that decides what status code each one maps to.
var (
	// ErrUnsupportedOp means the operation is not one this calculator knows.
	ErrUnsupportedOp = errors.New("unsupported operation")

	// ErrInvalidOperandLen means the operand count does not match the operation.
	ErrInvalidOperandLen = errors.New("invalid number of operands")

	// ErrInvalidOperand means an operand is not a finite number. JSON has no NaN or
	// Infinity literal, but "1e400" is valid JSON and decodes to +Inf.
	ErrInvalidOperand = errors.New("operand is not a finite number")

	// ErrDivisionByZero means a division had a zero divisor.
	ErrDivisionByZero = errors.New("division by zero")

	// ErrNegativeSqrt means a square root was asked of a negative number.
	ErrNegativeSqrt = errors.New("square root of a negative number")

	// ErrOverflow means the result left the range float64 can represent.
	ErrOverflow = errors.New("result overflows the representable range")

	// ErrUndefinedResult means the operation has no defined result for these operands.
	ErrUndefinedResult = errors.New("result is undefined")
)
