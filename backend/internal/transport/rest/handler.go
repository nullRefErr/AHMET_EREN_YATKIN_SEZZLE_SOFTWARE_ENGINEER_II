package rest

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"calculator/internal/domain"
	"calculator/internal/service"
)

// errNullOperand is returned when the body carries a JSON null where a number belongs.
var errNullOperand = errors.New("operand is null")

// calculateRequest is the wire shape of a calculation request.
//
// Operands are pointers so that a null can be told apart from a zero. Note what is not
// here: a "required" binding rule. That rule treats Go's zero value as absent, so it
// would reject "0 + 0" — a valid request and the first edge case anyone tries. Operand
// counts are checked by the domain instead, which keeps one rule in one place.
type calculateRequest struct {
	Operation string     `json:"operation"`
	Operands  []*float64 `json:"operands"`
}

func (r calculateRequest) toDomain() (domain.Request, error) {
	operands := make([]domain.Number, 0, len(r.Operands))
	for _, operand := range r.Operands {
		if operand == nil {
			return domain.Request{}, errNullOperand
		}
		operands = append(operands, *operand)
	}
	return domain.Request{
		Operation: domain.Operation(r.Operation),
		Operands:  operands,
	}, nil
}

// calculationResponse is the wire shape of a successful calculation.
type calculationResponse struct {
	Operation string          `json:"operation"`
	Operands  []domain.Number `json:"operands"`
	Result    domain.Number   `json:"result"`
	Cached    bool            `json:"cached"`
}

type operationResponse struct {
	Name     string `json:"name"`
	Operands int    `json:"operands"`
}

// handler translates HTTP into calculations and back. It performs no arithmetic of its
// own: everything it knows how to do is convert, delegate and map the answer.
type handler struct {
	calculator *service.Calculator
}

func (h handler) createCalculation(c *gin.Context) {
	var body calculateRequest

	// ShouldBindJSON rather than MustBindWith: the latter writes its own 400 and skips
	// the error envelope every other failure uses.
	if err := c.ShouldBindJSON(&body); err != nil {
		writeError(c, apiError{http.StatusBadRequest, codeInvalidRequest,
			"The request body must be JSON with an operation and a list of numeric operands."})
		return
	}

	request, err := body.toDomain()
	if err != nil {
		writeError(c, apiError{http.StatusBadRequest, codeInvalidRequest, "Every operand must be a finite number."})
		return
	}

	result, err := h.calculator.Calculate(c.Request.Context(), request)
	if err != nil {
		writeError(c, describe(err))
		return
	}

	c.JSON(http.StatusOK, calculationResponse{
		Operation: string(result.Calculation.Operation),
		Operands:  result.Calculation.Operands,
		Result:    result.Calculation.Result,
		Cached:    result.Cached,
	})
}

// listOperations lets the frontend learn the keypad instead of hardcoding a second copy
// of the operation list.
func (h handler) listOperations(c *gin.Context) {
	operations := domain.Operations()
	out := make([]operationResponse, 0, len(operations))

	for _, operation := range operations {
		count, ok := domain.OperandCount(operation)
		if !ok {
			continue
		}
		out = append(out, operationResponse{Name: string(operation), Operands: count})
	}

	c.JSON(http.StatusOK, gin.H{"operations": out})
}

// health answers whether the process is alive. It deliberately checks nothing else: this
// service has no dependency whose absence should take it out of rotation.
func (h handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
