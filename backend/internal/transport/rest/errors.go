package rest

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"calculator/internal/domain"
)

// The codes the API reports. Clients branch on these rather than on the message text,
// which is free to change or be translated.
const (
	codeInvalidRequest      = "INVALID_REQUEST"
	codeInvalidOperandCount = "INVALID_OPERAND_COUNT"
	codeDivisionByZero      = "DIVISION_BY_ZERO"
	codeNegativeSqrt        = "NEGATIVE_SQRT"
	codeNumericOverflow     = "NUMERIC_OVERFLOW"
	codeUndefinedResult     = "UNDEFINED_RESULT"
	codeInternalError       = "INTERNAL_ERROR"
)

// errorResponse is the one shape every failure takes.
type errorResponse struct {
	Error     errorDetail `json:"error"`
	RequestID string      `json:"request_id,omitempty"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// apiError is the outcome of mapping a domain error: what the client is told and with
// which status. Grouping the three keeps describe single-valued and writeError to two
// parameters.
type apiError struct {
	Status  int
	Code    string
	Message string
}

// describe maps a domain error onto the status, code and message the API reports.
//
// This is the only place that mapping is made. Adding an error to the domain means adding
// one case here, and nothing else in the transport layer has to know about it.
//
// The split between 400 and 422 is deliberate: 400 means the request could not be
// understood, 422 means it was understood but is not something arithmetic can answer.
func describe(err error) apiError {
	switch {
	case errors.Is(err, domain.ErrUnsupportedOp):
		return apiError{http.StatusBadRequest, codeInvalidRequest,
			"That operation is not supported. Call GET /api/v1/operations for the list."}
	case errors.Is(err, domain.ErrInvalidOperandLen):
		return apiError{http.StatusBadRequest, codeInvalidOperandCount,
			"That operation was given the wrong number of operands."}
	case errors.Is(err, domain.ErrInvalidOperand):
		return apiError{http.StatusBadRequest, codeInvalidRequest,
			"Every operand must be a finite number."}
	case errors.Is(err, domain.ErrDivisionByZero):
		return apiError{http.StatusUnprocessableEntity, codeDivisionByZero,
			"Division by zero is not defined."}
	case errors.Is(err, domain.ErrNegativeSqrt):
		return apiError{http.StatusUnprocessableEntity, codeNegativeSqrt,
			"The square root of a negative number is not a real number."}
	case errors.Is(err, domain.ErrOverflow):
		return apiError{http.StatusUnprocessableEntity, codeNumericOverflow,
			"The result is larger than this calculator can represent."}
	case errors.Is(err, domain.ErrUndefinedResult):
		return apiError{http.StatusUnprocessableEntity, codeUndefinedResult,
			"That calculation has no defined result."}
	default:
		return apiError{http.StatusInternalServerError, codeInternalError,
			"Something went wrong on our side."}
	}
}

// writeError ends the request with the standard envelope.
//
// Internal detail never reaches the body: the client gets a code, a sentence and the
// request id, and the matching log line holds everything else.
func writeError(c *gin.Context, e apiError) {
	c.AbortWithStatusJSON(e.Status, errorResponse{
		Error:     errorDetail{Code: e.Code, Message: e.Message},
		RequestID: requestIDFrom(c),
	})
}
