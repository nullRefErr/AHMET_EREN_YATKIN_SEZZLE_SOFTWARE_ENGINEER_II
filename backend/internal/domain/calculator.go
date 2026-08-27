// Package domain holds the calculation rules. It depends on nothing but the standard
// library: no web framework, no transport types, no storage. That is what keeps it
// testable, and it is enforced by review (see CLAUDE.md §3.1).
package domain

import "math"

// Number is the numeric type of every operand and result.
//
// float64 follows IEEE-754, so 0.1 + 0.2 is 0.30000000000000004. That is a property of
// the format, not a defect. Financial precision would need a decimal type; this alias is
// the single place such a change would begin.
type Number = float64

// Operation names a calculation this package can perform.
type Operation string

// The supported operations. Their wire names are these string values, so changing one is
// a breaking API change.
const (
	OpAdd        Operation = "add"
	OpSubtract   Operation = "subtract"
	OpMultiply   Operation = "multiply"
	OpDivide     Operation = "divide"
	OpPower      Operation = "power"
	OpSqrt       Operation = "sqrt"
	OpPercentage Operation = "percentage"
)

// Request is one calculation to perform.
type Request struct {
	Operation Operation
	Operands  []Number
}

// operationSpec pairs an operation with the number of operands it takes.
type operationSpec struct {
	operation    Operation
	operandCount int
}

// specs is the single source of truth for which operations exist and how many operands
// each one takes. Both OperandCount and Operations read from it, so adding an operation
// means editing one slice and one switch case.
var specs = []operationSpec{
	{OpAdd, 2},
	{OpSubtract, 2},
	{OpMultiply, 2},
	{OpDivide, 2},
	{OpPower, 2},
	{OpSqrt, 1},
	{OpPercentage, 2},
}

// OperandCount reports how many operands op takes. The second return value is false when
// op is not a supported operation.
func OperandCount(op Operation) (int, bool) {
	for _, s := range specs {
		if s.operation == op {
			return s.operandCount, true
		}
	}
	return 0, false
}

// Operations lists the supported operations in a stable order. The frontend reads this
// through the API instead of keeping its own copy.
func Operations() []Operation {
	ops := make([]Operation, 0, len(specs))
	for _, s := range specs {
		ops = append(ops, s.operation)
	}
	return ops
}

// Calculate performs req and returns its result.
//
// It is a pure function: the same request always produces the same answer, and it has no
// dependencies to inject or fake. That is why the service layer calls it directly rather
// than through an interface (SPEC.md D-15).
func Calculate(req Request) (Number, error) {
	wantOperands, ok := OperandCount(req.Operation)
	if !ok {
		return 0, ErrUnsupportedOp
	}
	if len(req.Operands) != wantOperands {
		return 0, ErrInvalidOperandLen
	}
	for _, operand := range req.Operands {
		if math.IsNaN(operand) || math.IsInf(operand, 0) {
			return 0, ErrInvalidOperand
		}
	}

	var result Number

	switch req.Operation {
	case OpAdd:
		result = req.Operands[0] + req.Operands[1]
	case OpSubtract:
		result = req.Operands[0] - req.Operands[1]
	case OpMultiply:
		result = req.Operands[0] * req.Operands[1]
	case OpDivide:
		// Guarded here so that 0/0 reports the division, not the NaN it would produce.
		if req.Operands[1] == 0 {
			return 0, ErrDivisionByZero
		}
		result = req.Operands[0] / req.Operands[1]
	case OpPower:
		result = math.Pow(req.Operands[0], req.Operands[1])
	case OpSqrt:
		if req.Operands[0] < 0 {
			return 0, ErrNegativeSqrt
		}
		result = math.Sqrt(req.Operands[0])
	case OpPercentage:
		// "a percent of b" — see SPEC.md D-04 for why this reading was chosen.
		result = req.Operands[0] * req.Operands[1] / 100
	default:
		// Reached only if an operation is added to specs without a case here.
		return 0, ErrUnsupportedOp
	}

	// One check for the whole switch rather than one per case: every operation can
	// overflow, and repeating the guard eight times is eight places to forget it.
	if math.IsInf(result, 0) {
		return 0, ErrOverflow
	}
	if math.IsNaN(result) {
		return 0, ErrUndefinedResult
	}
	return result, nil
}
